# LaunchDarkly CLI Client/Shell

This is very hacky and has no error handling.
Features:
- CRD Projects/Environments
- CRUD Flags sort of
- Lots of selection/autocomplete
- Connect to different servers with different keys
- Pages tables (fancy tables)
- auditlog


## Configuration

Right now we use expect a `~/.config/ldc.json` that looks like:

```
{
  "staging": {
    "apiToken": "your api token",
    "server": "https://app.launchdarkly.com/api/v2",
    "defaultProject": "default",
    "defaultEnvironment": "production"
  }
}
```

## Running 

Run `./run.sh`.

## Commands

Supported commands are:

```
creds
environments
clear
pwd
switch
flags
projects
log
exit
```
