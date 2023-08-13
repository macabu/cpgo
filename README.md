# cpgo - Continuous PGO

Your continuous profile-guided optimization assistant! Read more about PGO and Go [here](https://go.dev/doc/pgo)
  
This is a simple tool that enables you to keep a continuous feedback loop between your production code and profile-guided optimizations, in the following steps:
1. Scrape the CPU profile HTTP endpoint for your backends, generating a .pprof file (which stays in memory);
2. Search for relevant existing PGO profiles in the git repository of your backend;
3. If found, it will then merge the profiles;
4. Finally, it opens a Pull Request, updating the PGO profile.

*Disclaimer: this tool is more in a proof-of-concept/reference state than production-ready.*

## Getting Started

### Flags
You can run `go run ./cmd/cpgo -h` to view the available runtime flags:
```sh
Usage of cpgo:
  -configPath string
        The path (to) including the name of the config file with extension. Defaults to: ./config.yaml (default "./config.yaml")
  -githubToken string
        The Github token to be able to read the repositories and create the pull requests
  -verbose
        Whether to log debug messages
```

### Configuration File
There is also a configuration file available to set up all the backends you'd want to crawl, profile and then update its PGO file.

Here is a sample (also available in the repo as `config.sample.yaml`):
```yaml
---
# A list of backends, all properties below are mandatory for proper functioning.
backends:
  # HTTP endpoint to the CPU profiling handler, including the seconds
  # Make sure that the seconds match for the same existing profile.
- url: http://localhost:6060/debug/pprof/profile?seconds=30
  # Cron schedule, how often to run the above endpoint and update the profile. Reference: https://crontab.guru/
  schedule: '* * * * *'
  open_pull_request:
    # The full repo name, currently only supports GitHub.
    repository: http://github.com/my-org/my-repo
    # By default, the code will search for another existing file under the `default.pgo` name in your repo.
    # This is so we can take the new profile and merge it with the existing one.
    target_file: default.pgo
    # Finally the branch you want to target when the Pull Request is created.
    target_branch: main
```

### Running & Deploying
After configuring the `config.yaml` file, you can try `GITHUB_TOKEN=your-token make run` to try out the tool.
  
For production-use (aiming more towards containerization), a sample `Dockerfile` is provided.

### Profiling Application & Enabling PGO
1. Follow this guide to start profiling your application: https://pkg.go.dev/net/http/pprof
  - For PGO, a CPU profile is needed.
2. Build your Go program with `-pgo=auto` or follow the guide in this article: https://go.dev/doc/pgo

## Not Implemented (yet)
Non-exhaustive list of yet to be implemented features.

- Distribute binary and proper Docker file for ease of deployment;
- Creating pull requests in another Git Forge;
- Handling permanent vs transitory errors (it will always retry);
- Read GitHub token directly from env alternatively?;
- Support GitHub Apps instead of raw token auth;

Contributions are welcome!
