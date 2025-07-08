# Broadcom RoCE exporter

## About 
This is a Broadcom/RoCE exporter that reads all of the counters and gauges from
`/sys/class/infiniband/<bnxt_reX>/ports/1/hw_counters`.

* Counters are values that increase over time, starting with at 0 and incrementing, such as `tx_bytes`. 
* Gauges are values that can increase or decrease over time. They reflect the value at that point in time such as
  `active_qps`.
* The exporter is served out of port 9101. If you'd like a different port, you can update the last two lines at the
bottom of `main.go` with a new value, otherwise you can select your port if you are using Docker. 

## Install Go
Verify that you you have Go 1.20+ on your system with `go version`. 

* If you need to install Go, run these commands:

    ```
    wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    go version
    ```
 
* If you have an older version of Go and need to upgrade:
    
    ```
    apt remove golang-go -y
    rm -rf /usr/local/go
    wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    go version
    ```

## Create exporter binary
Either `git clone` this repo or create a directory such as `broadcom_exporter`.

    mkdir broadcom_exporter
    cd broadcom_exporter
    go mod init broadcom_exporter

Then run `vi main.go` and copy main.go from this repo into your directory. 

## Build
    go build -o broadcom_exporter
    
If you get a `required module for prometheus is needed` error, then run the following commands to install: 

    go get github.com/prometheus/client_golang/prometheus
    go get github.com/prometheus/client_golang/prometheus/promhttp

Then re-run the ```go build -o broadcom_exporter``` command. 

    ./broadcom_exporter

The exporter is served out of port 9101. 

## Run in a Docker container 
If you are building this for CentOS, use the Dockerfile called ```Dockerfile_centos```, but remove `_centos` from the
filename before running so it just says `Dockerfile`. 

If you did a `git clone` to copy the directory structure, you'll already have `Dockerfile` present. If you copied
`main.go` from this repo, then copy `Dockerfile` into same directory as `main.go`. 

    docker build -t broadcom-exporter .
    docker run -d --name broadcom-exporter -p 9101:9101 --restart always --privileged -v /sys:/sys:ro broadcom-exporter 
    docker ps
    curl localhost:9101/metrics
