package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/analytics/v3"
)

var (
	credsfile = "./config/credentials.json"

	startDate   = os.Getenv("START_DATE")
	scrapePort  = os.Getenv("SCRAPE_PORT")
	viewID      = os.Getenv("VIEW_ID")
	metrics     = os.Getenv("VIEW_METRICS")
	interval, _ = strconv.Atoi(os.Getenv("INTERVAL"))

	gauges = map[string]prometheus.Gauge{}
)

func main() {
	creds := getCreds(credsfile)

	jwtc := jwt.Config{
		Email:        creds["client_email"],
		PrivateKey:   []byte(creds["private_key"]),
		PrivateKeyID: creds["private_key_id"],
		Scopes:       []string{analytics.AnalyticsReadonlyScope},
		TokenURL:     creds["token_uri"],
	}

	httpClient := jwtc.Client(oauth2.NoContext)
	as, err := analytics.New(httpClient)
	if err != nil {
		panic(err)
	}

	rts := analytics.NewDataRealtimeService(as)
	gas := analytics.NewDataGaService(as)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%s", scrapePort), nil)

	for {
		col := make(map[string][]string)

		for _, metric := range strings.Split(metrics, ",") {
			parts := strings.Split(metric, ":")

			if col[parts[0]] == nil {
				col[parts[0]] = []string{metric}
			} else {
				col[parts[0]] = append(col[parts[0]], metric)
			}
		}

		for prefix, colMetrics := range col {
			for _, metric := range colMetrics {
				var value int

				switch prefix {
				case "ga":
					dt := time.Now()
					req := gas.Get(viewID, startDate, dt.Format("2006-01-02"), metric)

					res, err := req.Do()
					if err != nil {
						panic(err)
					}

					value, _ = strconv.Atoi(res.TotalsForAllResults[metric])

					break
				case "rt":
					req := rts.Get(viewID, metric)

					res, err := req.Do()
					if err != nil {
						panic(err)
					}

					value, _ = strconv.Atoi(res.TotalsForAllResults[metric])

					break
				}

				if gauges[metric] == nil {
					gauges[metric] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: fmt.Sprintf("ga_%s", metric),
						Help: "The number of " + metric,
					})
				}

				gauges[metric].Set(float64(value))
			}
		}

		time.Sleep(time.Second * time.Duration(interval))
	}
}

func getCreds(filename string) (r map[string]string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(data, &r); err != nil {
		panic(err)
	}

	return r
}
