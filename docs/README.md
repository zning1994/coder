# Coder

[!["GitHub Discussions"](https://img.shields.io/badge/%20GitHub-%20Discussions-gray.svg?longCache=true&logo=github&colorB=purple)](https://github.com/coder/coder/discussions) [!["Join us on Slack"](https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen)](https://coder.com/community) [![Twitter Follow](https://img.shields.io/twitter/follow/CoderHQ?label=%40CoderHQ&style=social)](https://twitter.com/coderhq) [![codecov](https://codecov.io/gh/coder/coder/branch/main/graph/badge.svg?token=TNLW3OAP6G)](https://codecov.io/gh/coder/coder)

Provision remote development environments with Terraform.

## Highlights

- Automate development environments for Linux, Windows, and MacOS in your cloud
- Start writing code with a single command
- Use one of many [examples](./examples) to get started

## Quickstart

Coder has two key concepts: *templates* and *workspaces*. Once templates are added in your deployment, users can create workspaces and start coding.

1. Install [the latest release](https://github.com/coder/coder/releases).

2. To tinker, start with dev-mode (all data is in-memory, and is destroyed on exit):

    ```bash
    coder start --dev
    ```

3. In a new terminal, create a new template (eg. Develop in Linux on Google Cloud):

    ```sh
    coder templates init
    cd <template-name>
    coder templates create
    ```

4. Create a new workspace and SSH in:

    ```sh
    coder workspaces create my-first-workspace
    coder ssh my-first-workspace
    ```

Under the hood, templates are Terraform code. Make changes to templates if necessary, or stick to the production-ready examples. 

```sh
cd <template-name>
# edit the template
vim main.tf
coder templates update <template-name>
```

Coder keeps your fleet of workspaces up-to-date and in-sync.

## Documentation

- [About Coder](./about)
   - [Architecture](./comparison.md)
   - [Coder vs. (other tool)](./comparison.md)
- [Installation](./installation)
- [Web UI vs. CLI](./web-cli.md)
- [Templates](./templates)
   - [Persistant vs. ephemeral](./templates/state.md)
   - [Troubleshooting](./templates/troubleshooting.md)
- [Workspaces](./workspaces)
  - [Supported IDEs](./workspaces/IDEs.md)
- [Users & Organizations](./users)
  - [Roles](./users/roles.md)
  - [SSH keys (for git)](./users/dotfiles.md)
  - [Dotfiles](./users/dotfiles.md) 

## Contributing

Read the [contributing docs](./CONTRIBUTING.md).

