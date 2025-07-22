# relish-notifier

A Go application that uses browser automation to tell you when your Relish order has arrived.

## Usage

```
Monitor Relish orders and send notifications.

Credentials are retrieved from the system keychain (service: relish-notifier, accounts: EMAIL/PASSWORD).
If the keychain is unavailable, the environment variables `RELISH_USERNAME` and
`RELISH_PASSWORD` will be used as fallback.

Usage:
  relish-notifier [flags]

Flags:
  -i, --check-interval int      How often to check for delivery (seconds) (default 30)
  -c, --command string          Run this command when your order has arrived
      --extensions              Enable browser extensions (default true)
      --headless                Run Chrome in headless mode (default true)
  -h, --help                    help for relish-notifier
      --once                    Check once and exit
  -t, --page-timeout duration   Set page timeout (default 10s)
  -v, --verbose count           Increase verbosity (-v: info, -vv: debug)
      --version                 version for relish-notifier
```

## Relish credentials

Credentials are stored in the system keyring and can be set via the following methods:

### Using Go and the keyring library:

```bash
$ go run -c '
import "github.com/zalando/go-keyring"
keyring.Set("relish-notifier", "EMAIL", "<your email>")
keyring.Set("relish-notifier", "PASSWORD", "<your password>")
'
```

### Using Python (if available):

```bash
$ python
Python 3.12.10 (main, Apr 22 2025, 00:00:00) [GCC 14.2.1 20240912 (Red Hat 14.2.1-3)] on linux
Type "help", "copyright", "credits" or "license" for more information.
>>> import keyring
>>> keyring.set_password("relish-notifier", "EMAIL", "<your email>")
>>> keyring.set_password("relish-notifier", "PASSWORD", "<your password>")
>>> exit()
```

### Using environment variables:

As a fallback, you can set environment variables:

```bash
export RELISH_USERNAME="<your email>"
export RELISH_PASSWORD="<your password>"
```

## Installation

### From source:

1. Clone this repository.

2. From inside the repository:

    ```bash
    go build -o relish-notifier
    ```

3. Optionally, install to your PATH:

    ```bash
    go install
    ```

## License

relish-notifier -- get notified when your lunch arrives

Copyright (C) 2025 Lars Kellogg-Stedman

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
