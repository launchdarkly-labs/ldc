> [!CAUTION]
> This repository has been superceded by [launchdarkly/ldcli](https://github.com/launchdarkly/ldcli).
>
> The project in this repository remains unsupported and is no longer going to accept contributions effective immediately.
>
> The prior readme contents remain accessible below for posterity.

# LDC - LaunchDarkly CLI/Console

[![GolangCI](https://golangci.com/badges/github.com/golangci/golangci-lint.svg)](https://golangci.com)

*This is BETA software until version 1.0 and the interface may change.*

This command-line interface provides basic interaction with LaunchDarkly. You can list and operate on projects, environments, flags, and metrics.

## Configuring

You can specify configuration information in the `./ldc.json` file. This sets the default project and environment information for where your commands will operate.

The format is:

```json
{
    "<your configuration name>": {
        "apitoken": "<your api token>",
        "defaultenvironment": "<your environment key>",
        "defaultproject": "<your project key>",
        "server": "<your server, optional, defaults to https://app.launchdarkly.com>"
    }
}
```

You can create an API access token from the [**Account settings**](https://app.launchdarkly.com/settings) page in the LaunchDarkly application, on the **Authorization** tab.

## Running

To run a single command, use:

```
./run.sh <command>
```

To run a series of commands in an interactive shell, use:

```
./run.sh shell
```

## Commands

The supported top-level commands are:

* `clear`: Clear the screen
* `configs`: Update configuration information
  * Available actions are `add`, `edit`, `rename`, `rm` (remove), `set` (change which configuration you're using)
* `environments`: List and operate on environments
  * Available actions are `list`, `show`, `create`, `delete`
* `exit`: Exit the program
* `flags`: List and operate on flags
  * Available actions are: `list` (default), `show`, `create`, `create-toggle`, `add-tag`, `remove-tag`, `on`, `off`, `rollout`, `fallthrough`, `edit`, `delete`, `status`
* `goals`: List and operate on metrics
  * Available actions are `list`, `create`, `show`, `results`, `attach`, `detach`, `edit`, `delete`
* `help`: Display help
* `json`: Set JSON mode
* `log`: Search audit log entries
* `projects`: List and operate on projects
  * Available actions are `list`, `show`, `create`, `delete`
* `pwd`: Show current configuration context
* `shell`: Run shell
* `switch`: Switch to a given project and environment
* `token`: Set API token
* `version`: Show version

For commands that have associated actions, use the format:

```
<command> <action> <arguments>
```

For example, to display information about a particular flag, use:

```
./run.sh flags show <your flag key>
```

Similarly, to create a new flag in the current project, use:

```
./run.sh flags create <new flag key>
```

The current project is the default project specified in your config, unless you've used the `configs` command to change which configuration you're using.

## Referencing resources

By default, commands apply to resources in the current project and environment. To apply a command to a resource in a different project or environment, you can specify a path to the resource.

A full path to a resource looks like this:

```
//<config>/<project>/<resource>
```

You can also specify an absolute resource path:

```
# Enable a flag using absolute syntax /<project>/<resource>
./run.sh flags /my-project/my-env/my-flag toggle on
```

Finally, you can reference the default project or environment with the special `...` key:

```
# Enable a flag using the default project syntax "/.../<resource>"
./run.sh flags /.../my-env/my-flag toggle on

# Enable a flag using the default project and environment syntax "/.../.../<resource>"
./run.sh flags /.../.../my-flag toggle on

# Enable a flag for a specific config using the default project and environment syntax "/.../.../<resource>"
./run.sh flags //my-config/.../.../my-flag toggle on
```
