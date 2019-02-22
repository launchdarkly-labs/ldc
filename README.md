# LDC - LaunchDarkly CLI/Console

[![CircleCI](https://circleci.com/gh/launchdarkly/ldc.svg?style=svg)](https://circleci.com/gh/launchdarkly/ldc)

[![GolangCI](https://golangci.com/badges/github.com/golangci/golangci-lint.svg)](https://golangci.com)

*This is BETA software until version 1.0 and the interface may change.*
 
## Configuration


You may specific `~/.config/ldc.json` that looks like:

```
{
  "staging": {
    "token": "<your api token>",
    "server": "https://app.launchdarkly.com/",
    "defaultProject": "default",
    "defaultEnvironment": "production"
  }
}
```

## Examples

A simple example is:

```
ldc flags create <new flag>
```

This will create a flag in the current project (the default project for your config unless you've changed your project)

## Referencing resources


Commands can take resource definitions as relative, meaning they apply to the current project and environment (if applicable).

A full path to a resource looks like:

```
//<config>/<project>/<resource>
```

It is also possible to use absolute paths relative to the currently selected config:

```
# Enable a flag using absolute syntax /<project>/<resource>
ldc flags /my-project/my-env/my-flag toggle on

```

Finally, you can reference the default project or environment with the special `...` key:

```
# Enable a flag using the default project syntax "/.../<resource>"
ldc flags /.../my-env/my-flag toggle on

# Enable a flag using the default project and environment syntax "/.../.../<resource>"
ldc flags /.../.../my-flag toggle on

# Enable a flag for a specific config using the default project and environment syntax "/.../.../<resource>"
ldc flags //my-config/.../.../my-flag toggle on
```

 
## Running 

Run `./run.sh`.

## Commands

Supported top-level commands are:

```
creds
environments
clear
pwd
switch
flags
goals 
projects
log
exit
```
