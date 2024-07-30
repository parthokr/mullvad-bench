### Download the executable from release

### Usage
* Search across all servers
```bash
./mullvad-bench
```
This will ping all servers and save the output to **bench_results.csv**.
* Search across servers in a specific country
```bash
./mullvad-bench -c <country-code>
```
Example: `./mullvad-bench -c sg`
* List all available countries
```bash
./mullvad-bench -lc
```
* Set a timeout for ping
```bash
./mullvad-bench -t <timeout>
```
Example: `./mullvad-bench -t 1s`

* Specify output filename
```bash
./mullvad-bench -o <filename>
```
Example: `./mullvad-bench -o bench1.csv`

* Show help
```bash
./mullvad-bench -h
```