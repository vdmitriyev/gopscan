## About

`gopscan` is a simple command-line utility written in Go that allows to scan for open TCP ports on a list of servers and list of ports.

## Key Features

* **Server List from File:** 
    + Reads a list of target servers (hostnames or IP addresses) from a specified YAML file
* **Ports from Multiple Files:** 
    + Reads port numbers to scan from one or more files within a specified directory. Each file should contain a comma-separated list of ports.
* **Concurrent Scanning:** 
    - Should utilizes Go's concurrency features (goroutines) to scan multiple ports on multiple servers simultaneously, making the process efficient.
* **Structured Logging:** 
    - Uses the `zap` package for structured logging of open ports to a dedicated file. Each log entry includes the hostname/IP address and the open port number.
* **Report:** 
    - Generates reports and sends them by email.

## Usage


1.  **Prepare input files:**
    * Create a `servers.yaml` file (or the file specified by `-servers`). See example of the file below.
    * Create a `ports` directory (or the directory specified by `-portsdir`) containing one or more files with comma-separated port numbers.    


1.  **Build the utility:**
    ```bash
    go build main.go
    ```
    This will create an executable file (e.g., `gopscan` on Linux/macOS, `gopscan.exe` on Windows).

1.  **Run the utility:**
    ```bash
    ./gopscan
    ```
    You can also specify different input files and directories using the flags:
    ```bash
    ./gopscan -servers servers.yaml -portsdir port
    ```

1.  **Check the output:**
    * The utility will print progress information to the console.
    * Any open ports found will be logged in the `gopscan.log` file in the same directory where you run the tool.

1. **Sending emails:**
    * There is a possibility to send reports of open ports as an email.
    * Create a `.env` file with email settings. See example of the file below.


## Format Example: `servers.yaml` 

An example of `servers.yaml` file

```yaml
servers:
  - name: webserver01
    allowedPorts:
      - 80
      - 443
      - 8080
  - name: dbserver01
    allowedPorts:
      - 5432
      - 3306
```

## Format Example: `.env` 

An example of `.env` file

```text
EMAIL_SERVER = '<CHANGE-ME>'
EMAIL_PORT = 587
EMAIL_USE_TLS = True
EMAIL_USERNAME = '<CHANGE-ME>'
EMAIL_PASSWORD = '<CHANGE-ME>'
EMAIL_SENDER = '<CHANGE-ME>'
EMAIL_ADMIN_NOTIFIER = '<CHANGE-ME>'
EMAIL_READONLY_MODE = True
```

## Data

* Collection of ports as a files:
    - https://github.com/HeckerBirb/top-nmap-ports-csv

## License

[MIT](LICENSE)