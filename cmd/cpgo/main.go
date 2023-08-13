package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/pprof/profile"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/macabu/cpgo/internal/config"
	"github.com/macabu/cpgo/internal/flags"
	"github.com/macabu/cpgo/internal/gitops"
	"github.com/macabu/cpgo/internal/gitops/gh"
	"github.com/macabu/cpgo/internal/pprof"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	flags, helpOutput, err := flags.Parse(os.Args[0], os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Println(helpOutput)

			os.Exit(1)
		}

		log.Fatal().Err(err).Msg("Could not parse flags")
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if flags.LogVerbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	ghClient := gh.NewClientWithAccessToken(ctx, flags.GithubToken)

	cfg, err := config.Parse(flags.ConfigPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse config")
	}

	s := gocron.NewScheduler(time.UTC)
	s.SetMaxConcurrentJobs(runtime.NumCPU()-1, gocron.WaitMode)

	for i, backend := range cfg.Backends {
		_, err := s.Cron(backend.Schedule).Do(func() {
			if err := run(ctx, backend, ghClient); err != nil {
				log.Error().Err(err).Str("backend_url", backend.URL).Str("repo", backend.OpenPR.Repo).Msg("Failed to process backend")
			}
		})
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to schedule run")
		}

		log.Info().Str("backend_url", backend.URL).Str("repo", backend.OpenPR.Repo).Msgf("[%v] Scheduled job!", i+1)
	}

	go func() {
		<-sigChan

		cancel()
		s.Stop()

		log.Debug().Msg("Stopping schedules, and going to sleep. Bye bye!")
	}()

	s.StartBlocking()
}

func run(ctx context.Context, backend config.Backend, ghClient *gh.Client) error {
	ghRepo := gh.ParseRepoURL(backend.OpenPR.Repo)

	logger := log.With().Str("url", backend.URL).Str("repo_org", ghRepo.Org).Str("repo_name", ghRepo.Name).Logger()

	profileFetcher := pprof.NewFetcher(http.DefaultClient)

	logger.Debug().Msg("Fetching profile")

	newProfile, err := profileFetcher.FromURL(ctx, backend.URL)
	if err != nil {
		return fmt.Errorf("profileFetcher.FromURL: %w", err)
	}

	logger.Debug().Int64("profile_duration_ns", newProfile.DurationNanos).Msg("Profile fetched!")

	logger.Debug().Msg("Checking whether there is already another profile")

	opts := gh.Options{
		Repo:       ghRepo,
		Filename:   backend.OpenPR.TargetFile,
		MainBranch: backend.OpenPR.TargetBranch,
	}

	downloadURL, err := ghClient.ExistingPGOFileURL(ctx, opts)
	if err != nil && !errors.Is(err, gitops.ErrPGOFileNotFound) {
		return fmt.Errorf("ghClient.ExistingPGOFileURL: %w", err)
	}

	profiles := []*profile.Profile{newProfile}

	if downloadURL != "" {
		logger.Info().Str("download_url", downloadURL).Msg("Found existing PGO file. Downloading it...")

		existingProfile, err := profileFetcher.FromURL(ctx, downloadURL)
		if err != nil {
			return fmt.Errorf("profileFetcher.FromURL: %w", err)
		}

		profiles = append(profiles, existingProfile)
	}

	var b bytes.Buffer

	if err := pprof.MergeProfiles(&b, profiles); err != nil {
		return fmt.Errorf("pprof.MergeProfiles: %w", err)
	}

	logger.Debug().Msg("Merged profiles!")

	prURL, err := ghClient.UpdatePGOFile(ctx, opts, b.Bytes())
	if err != nil {
		return fmt.Errorf("ghClient.UpdatePGOFile: %w", err)
	}

	logger.Info().Str("pr_url", prURL).Msg("Created new PR")

	return nil
}
