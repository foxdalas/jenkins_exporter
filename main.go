package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		url           = kingpin.Flag("jenkins.url", "Jenkins server url.").Default("http://localhost:8080").String()
		username      = kingpin.Flag("jenkins.username", "Jenkins server username.").Default("admin").String()
		password      = kingpin.Flag("jenkins.password", "Jenkins server token/password.").Default("admin").String()
		timeout       = kingpin.Flag("jenkins.timeout", "Jenkins server connect timeout.").Default("30s").Duration()
		runInterval   = kingpin.Flag("run.interval", "Exporter collect metrics interval").Default("5m").Duration()
		listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9118").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("jenkins_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting jenkins_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter := NewExporter(*url, *username, *password, *runInterval, *timeout)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
      <head><title>Jenkins Exporter</title></head>
      <body>
      <h1>Jenkins Exporter</h1>
      <p><a href='` + *metricsPath + `'>Metrics</a></p>
      </body>
      </html>`))
	})

	log.Infoln("Starting HTTP server on", *listenAddress)
	srv := http.Server{
		Addr:    *listenAddress,
		Handler: http.DefaultServeMux,
	}
	exporter.Run()
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("http server failure")
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	rsig := <-sig
	log.Infoln("Received signal %s, going to shutdown", rsig.String())
	exporter.Stop()
	srv.Shutdown(context.Background())
}
