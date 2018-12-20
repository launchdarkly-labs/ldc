# LaunchDarkly CLI Client/Shell

*This is BETA software until version 1.0 and the interface may change.*
 
## Configuration


You may specific `~/.config/ldc.json` that looks like:

```
{
  "staging": {
    "token": "your api token",
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
