## About

`gopscan` is a simple command-line utility written in Go that allows you to scan for open TCP ports on a list of servers and IP address ranges.


## External Data

* https://github.com/HeckerBirb/top-nmap-ports-csv

## Key Features

* **Server List from File:** Reads a list of target servers (hostnames or IP addresses) from a specified text file, one server per line.
* **Ports from Multiple Files:** Reads port numbers to scan from one or more files within a specified directory. Each file should contain a comma-separated list of ports.
* **IP Address Range Scanning:** Reads IP address ranges from a CSV file (format: `start;stop`) and expands them into individual IP addresses for scanning.
* **Concurrent Scanning:** Utilizes Go's concurrency features (goroutines) to scan multiple ports on multiple servers simultaneously, making the process efficient.
* **Structured Logging:** Uses the `slog` package for structured logging of open ports to a dedicated file (`open_ports.log`). Each log entry includes the hostname/IP address and the open port number.
* **Command-Line Flags:** Provides flags to customize the input files and directories:
    * `-servers`: Path to the file containing the list of servers (default: `servers.csv`).
    * `-portsdir`: Path to the directory containing files with comma-separated ports (default: `ports`).
    * `-ranges`: Path to the file containing IP ranges (default: `ranges.csv`).

## Usage

1.  **Build the utility:**
    ```bash
    go build main.go
    ```
    This will create an executable file (e.g., `gopscan` on Linux/macOS, `gopscan.exe` on Windows).

2.  **Prepare input files:**
    * Create a `servers.txt` file (or the file specified by `-servers`) with one server per line.
    * Create a `ports` directory (or the directory specified by `-portsdir`) containing one or more files with comma-separated port numbers.
    * Create a `ranges.csv` file (or the file specified by `-ranges`) with IP address ranges in the format `start;stop`.

3.  **Run the utility:**
    ```bash
    ./gopscan
    ```
    You can also specify different input files and directories using the flags:
    ```bash
    ./gopscan -servers my_servers.txt -portsdir my_port_lists -ranges ip_ranges.csv
    ```

4.  **Check the output:**
    * The utility will print progress information to the console.
    * Any open ports found will be logged in the `open_ports.log` file in the same directory where you run the tool.

## Requirements

* Go version 1.18 or later (for `slog` package).

## License

This project is open-source and available under the [MIT License](LICENSE) (you might want to add a LICENSE file if you intend to open-source it).