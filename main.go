// Broadcom Exporter

package main

import (
        "fmt"
        "io/ioutil"
        "net/http"
        "os"
        "path/filepath"
        "strconv"
        "strings"
        "time"

        "github.com/prometheus/client_golang/prometheus"
        "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric definitions – all treated as gauges since values are point-in-time
var metricDefinitions = []struct {
        name string
        help string
}{
        {"active_ahs", "Active address handles"},
        {"active_cqs", "Active completion queues"},
        {"active_mrs", "Active memory regions"},
        {"active_mws", "Active memory windows"},
        {"active_pds", "Active protection domains"},
        {"active_qps", "Active queue pairs"},
        {"active_rc_qps", "Active reliable connection queue pairs"},
        {"active_srqs", "Active shared receive queues"},
        {"active_ud_qps", "Active unreliable datagram queue pairs"},

        {"watermark_ahs", "Maximum address handles"},
        {"watermark_cqs", "Maximum completion queues"},
        {"watermark_mrs", "Maximum memory regions"},
        {"watermark_mws", "Maximum memory windows"},
        {"watermark_pds", "Maximum protection domains"},
        {"watermark_qps", "Maximum queue pairs"},
        {"watermark_rc_qps", "Maximum reliable connection queue pairs"},
        {"watermark_srqs", "Maximum shared receive queues"},
        {"watermark_ud_qps", "Maximum unreliable datagram queue pairs"},

        {"bad_resp_err", "Bad response errors"},
        {"dup_req", "Duplicate requests encountered"},
        {"local_protection_err", "Local memory protection errors"},
        {"local_qp_op_err", "Local queue pair operational errors"},
        {"max_retry_exceeded", "Maximum retry limit exceeded"},
        {"mem_mgmt_op_err", "Memory management operation errors"},
        {"missing_resp", "Missing responses"},
        {"oos_drop_count", "Out-of-sequence drop count"},
        {"pacing_alerts", "Pacing alerts"},
        {"pacing_complete", "Pacing operations completed"},
        {"pacing_reschedule", "Pacing operations rescheduled"},
        {"recoverable_errors", "Recoverable errors"},
        {"remote_access_err", "Remote memory access errors"},
        {"remote_invalid_req_err", "Invalid remote requests"},
        {"remote_op_err", "Remote operation errors"},
        {"res_cmp_err", "Completion errors"},
        {"res_cq_load_err", "Completion queue load errors"},
        {"res_exceed_max", "Exceeded maximum resource limit"},
        {"res_exceeds_wqe", "Exceeded work queue element limit"},
        {"res_invalid_dup_rkey", "Invalid or duplicate remote key errors"},
        {"res_irrq_oflow", "Inbound request queue overflow"},
        {"res_length_mismatch", "Length mismatch errors"},
        {"res_mem_err", "Memory errors"},
        {"res_opcode_err", "Opcode errors"},
        {"res_rem_inv_err", "Remote invalid request errors"},
        {"res_rx_domain_err", "Receive domain errors"},
        {"res_rx_invalid_rkey", "Receive invalid remote key errors"},
        {"res_rx_no_perm", "Receive no permission errors"},
        {"res_rx_pci_err", "Receive PCI errors"},
        {"res_rx_range_err", "Receive range errors"},
        {"res_srq_err", "Shared receive queue errors"},
        {"res_srq_load_err", "Shared receive queue load errors"},
        {"res_tx_domain_err", "Transmit domain errors"},
        {"res_tx_invalid_rkey", "Transmit invalid remote key errors"},
        {"res_tx_no_perm", "Transmit no permission errors"},
        {"res_tx_pci_err", "Transmit PCI errors"},
        {"res_tx_range_err", "Transmit range errors"},
        {"res_unaligned_atomic", "Unaligned atomic operation errors"},
        {"res_unsup_opcode", "Unsupported opcode errors"},
        {"res_wqe_format_err", "Work queue element format errors"},
        {"resize_cq_cnt", "Completion queue resize events"},
        {"rnr_naks_rcvd", "Receiver not ready (RNR) NACKs received"},
        {"seq_err_naks_rcvd", "Sequence error NACKs received"},
        {"to_retransmits", "Timeout retransmissions"},

        {"rx_atomic_req", "Atomic operation requests received"},
        {"rx_bytes", "Total bytes received"},
        {"rx_cnp_pkts", "Congestion notification packets received"},
        {"rx_ecn_marked_pkts", "Packets marked with explicit congestion notification"},
        {"rx_good_bytes", "Total bytes received without errors"},
        {"rx_good_pkts", "Total packets received successfully"},
        {"rx_out_of_buffer", "Packets dropped due to buffer exhaustion"},
        {"rx_pkts", "Total packets received (all)"},
        {"rx_read_req", "Read requests received"},
        {"rx_read_resp", "Read responses received"},
        {"rx_roce_discards", "RoCE packets discarded"},
        {"rx_roce_errors", "Errors in received RoCE packets"},
        {"rx_roce_only_bytes", "Total bytes received via RoCE"},
        {"rx_roce_only_pkts", "Total packets received via RoCE"},
        {"rx_send_req", "Send requests received"},
        {"rx_write_req", "Write requests received"},

        {"tx_atomic_req", "Atomic operation requests transmitted"},
        {"tx_bytes", "Total bytes transmitted"},
        {"tx_cnp_pkts", "Congestion notification packets transmitted"},
        {"tx_pkts", "Total packets transmitted (all)"},
        {"tx_read_req", "Read requests transmitted"},
        {"tx_read_resp", "Read responses transmitted"},
        {"tx_roce_discards", "RoCE packets discarded during transmission"},
        {"tx_roce_errors", "Errors in transmitted RoCE packets"},
        {"tx_roce_only_bytes", "Total bytes transmitted via RoCE"},
        {"tx_roce_only_pkts", "Total packets transmitted via RoCE"},
        {"tx_send_req", "Send requests transmitted"},
        {"tx_write_req", "Write requests transmitted"},

        {"unrecoverable_err", "Unrecoverable errors"},
        {"db_fifo_register", "DB FIFO hardware register value"},
}

// Prometheus registry
var (
        customRegistry = prometheus.NewRegistry()
        metrics        = make(map[string]*prometheus.GaugeVec)
)

func registerMetrics() {
        for _, def := range metricDefinitions {
                metrics[def.name] = prometheus.NewGaugeVec(
                        prometheus.GaugeOpts{
                                Name: def.name,
                                Help: def.help,
                        },
                        []string{"nic"},
                )
                customRegistry.MustRegister(metrics[def.name])
        }
}

func discoverNICs() ([]string, error) {
        nicPaths, err := filepath.Glob("/sys/class/infiniband/*")
        if err != nil {
                return nil, fmt.Errorf("failed to list NIC paths: %w", err)
        }
        if len(nicPaths) == 0 {
                return nil, fmt.Errorf("no NICs found in /sys/class/infiniband/")
        }
        fmt.Printf("Discovered NICs: %v\n", nicPaths)
        return nicPaths, nil
}

func parseAndUpdateMetrics(nicPath string) error {
        nicName := filepath.Base(nicPath)
        hwCountersPath := filepath.Join(nicPath, "ports/1/hw_counters")

        if _, err := os.Stat(hwCountersPath); os.IsNotExist(err) {
                fmt.Printf("NIC %s does not have hw_counters, skipping...\n", nicName)
                return nil
        }

        for _, def := range metricDefinitions {
                counterFile := filepath.Join(hwCountersPath, def.name)
                data, err := ioutil.ReadFile(counterFile)
                if err != nil {
                        if os.IsNotExist(err) {
                                fmt.Printf("Metric file %s missing for NIC %s, skipping...\n", def.name, nicName)
                                continue
                        }
                        return fmt.Errorf("error reading %s: %w", counterFile, err)
                }

                valueStr := strings.TrimSpace(string(data))
                value, err := strconv.ParseFloat(valueStr, 64)
                if err != nil {
                        return fmt.Errorf("error parsing value from %s: %w", counterFile, err)
                }

                if gauge, exists := metrics[def.name]; exists {
                        gauge.WithLabelValues(nicName).Set(value)
                }
        }
        return nil
}

func updateMetrics(nicPaths []string) {
        for _, nicPath := range nicPaths {
                err := parseAndUpdateMetrics(nicPath)
                if err != nil {
                        fmt.Printf("Error updating metrics for NIC %s: %v\n", nicPath, err)
                }
        }
}

func main() {
        registerMetrics()

        nicPaths, err := discoverNICs()
        if err != nil {
                fmt.Printf("Error discovering NICs: %v\n", err)
                os.Exit(1)
        }

        go func() {
                for {
                        updateMetrics(nicPaths)
                        time.Sleep(15 * time.Second)
                }
        }()

        http.Handle("/metrics", promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}))
        fmt.Println("Serving metrics on :9101")
        http.ListenAndServe(":9101", nil)
}

