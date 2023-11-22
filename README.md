# DayLog

Keep track of what you're doing when you do it and forget trying to write summaries at EOD. DayLog[^1] is a dead-simple tool for terminal enthusiasts for taking quick notes throughout your day. It helps you quickly edit date-stamped markdown files where you can take notes about what you're doing, leave note in tomorrow's log for your future self

## Usage

To write or edit today's file, run `daylog` and today's log will be opened in `$EDITOR`.

To view today's file, run `daylog show`.

To interact with a past or future log supply a date (`daylog show -- 2023/01/07`), or a more casual realtive reference, "tomorrow", "yesterday", "1 day ago", etc.

### Log storage

Logs are stored in `$XDG_DATA_HOME/daylog`.

## Installation

### Install a prebuilt binary

1. Get the URL for the desired version and platform from the [releases page](https://github.com/notnmeyer/daylog-cli/releases).
2. Download the release,
    ```
    curl -LO https://github.com/notnmeyer/daylog-cli/releases/download/v0.0.3/daylog-cli_Darwin_arm64.tar.gz
    ```
    or
   ```
   wget https://github.com/notnmeyer/daylog-cli/releases/download/v0.0.3/daylog-cli_Darwin_arm64.tar.gz
   ```
4. Extract the tar file, `tar -xzvf daylog-cli_Darwin_arm64.tar.gz`.
5. Move the `daylog` binary where you want it—`mv daylog ~/bin`, `mv daylog /usr/local/bin/`, etc.

Or just copy and paste it all,

```shell
release_version=v0.0.3
installation_directory=~/bin/
cd $(mktemp -d)
curl -LO https://github.com/notnmeyer/daylog-cli/releases/download/$release_version/daylog-cli_Darwin_arm64.tar.gz
tar -xzvf daylog-cli_Darwin_arm64.tar.gz
mv daylog "$installation_directory"/
```

### Install from source

1. Build the project with, `go build -o ~/bin/daylog main.go`, substituting `~/bin/daylog` for a different path if you prefer.

[^1]: DayLog ah ahh ahhhhhh, fighter of the night log ah ahh ahhhhh.

    ![image](https://github.com/notnmeyer/daylog-cli/assets/672246/fa27a3ec-8044-4813-bfb0-3494eab97a98)

    DayyyyyyyyyyLLooooooooog!
    
    ![image](https://github.com/notnmeyer/daylog-cli/assets/672246/949b7eee-aa63-484a-a366-231462ac9563)
