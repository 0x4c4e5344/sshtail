# SSH Tail

> **Note**
> This is a fork of [drognisep/sshtail](https://github.com/drognisep/sshtail) project.

This is a CLI app that will set up SSH connections to multiple hosts specified in the given spec file using a key of your choice, tail the named file, and aggregate the output to the calling terminal's STDOUT.

**Note:** This utility uses the `tail` executable on the remote host to facilitate its base functionality. This limitation is mostly because I haven't figured out any other way yet. PRs welcome!

![Go](https://github.com/drognisep/sshtail/workflows/Go/badge.svg?branch=master)

## Installation
If you're [using Go 1.16+](https://blog.golang.org/go116-module-changes), run this.
```bash
go install github.com/0x4c4e5344/sshtail@latest
```

Otherwise, just run this command!
```bash
go get github.com/0x4c4e5344/sshtail
```

You can also download one of the releases directly.

## Examples
An example file can be output to "test.yml" by running
```bash
sshtail spec init --with-comments test.yml
```

Here's the output.
```yaml
# Hosts and files to tail
hosts:
  host1:
      hostname: remote-host-1
    # Excluding the username here will default it to the current user name
      file: /var/log/syslog
      identity_file: ~/.ssh/id_rsa
      # Default SSH port, can be excluded.
    port: 22
  host2:
      hostname: remote-host-2
      username: me
      identity_file: ~/.ssh/id_rsa
      file: /var/log/syslog
    port: 22
```

## Hosts
This section is used to specify the host machines to connect to. `hostname` and `file` are required, but `port` may be excluded if the default SSH port of 22 is desired.

The values of "host1" and "host2" can be anything you wish, and are primarily used to match a specified host with a given key path, and to tag the output to your terminal like so:

```
host1 | A line posted to /var/log/syslog on remote-host-1...
host1 | And another one...
```

## Common Commands
This will create a spec file useful for understanding the format, exactly like what is shown above.
```bash
sshtail spec init --with-comments <spec file name>
```

To make it a bit more useful, this will exclude the `keys` section for portability and won't print comments.
```bash
sshtail spec init --exclude-keys <spec file name>
```

Finally, to execute a spec use this command. If the configured key is encrypted then the user will be asked to enter their pass phrase each time it is referenced. This is for security purposes because I don't want to cache the pass phrase in memory.
```bash
sshtail spec run <spec file name>
```
