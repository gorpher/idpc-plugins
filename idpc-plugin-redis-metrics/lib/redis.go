package iredis

import (
	"flag"
	"fmt"
	"github.com/fzzy/radix/redis"
	"github.com/gorpher/go-idpc-plugin"
	"github.com/rs/zerolog/log"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type RedisPlugin struct {
	plugin.MetricsPlugin
	Host     string
	Port     string
	Password string
	Socket   string
	Key      string
}

func authenticateByPassword(c *redis.Client, password string) error {
	if r := c.Cmd("AUTH", password); r.Err != nil {
		log.Err(r.Err).Msg("Failed to authenticate.")
		return r.Err
	}
	return nil
}

func (r RedisPlugin) Metrics() (map[string]interface{}, error) {
	network := "tcp"
	target := fmt.Sprintf("%s:%s", r.Host, r.Port)
	if r.Socket != "" {
		target = r.Socket
		network = "unix"
	}
	c, err := redis.DialTimeout(network, target, time.Duration(5)*time.Second)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect redis. ")
		return nil, err
	}
	defer c.Close()

	if r.Password != "" {
		if err = authenticateByPassword(c, r.Password); err != nil {
			return nil, err
		}
	}

	rds := c.Cmd("info")
	if rds.Err != nil {
		log.Error().Err(rds.Err).Msg("Failed to run info command. ")
		return nil, rds.Err
	}
	str, err := rds.Str()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch information. ")
		return nil, err
	}

	stat := make(map[string]interface{})

	keysStat := 0.0
	expiresStat := 0.0
	var slaves []string

	for _, line := range strings.Split(str, "\r\n") {
		if line == "" {
			continue
		}
		if re, _ := regexp.MatchString("^#", line); re {
			continue
		}

		record := strings.SplitN(line, ":", 2)
		if len(record) < 2 {
			continue
		}
		key, value := record[0], record[1]

		if re, _ := regexp.MatchString("^slave\\d+", key); re {
			slaves = append(slaves, key)
			kv := strings.Split(value, ",")
			var offset, lag string
			if len(kv) == 5 {
				_, _, _, offset, lag = kv[0], kv[1], kv[2], kv[3], kv[4]
				lagKv := strings.SplitN(lag, "=", 2)
				lagFv, err := strconv.ParseFloat(lagKv[1], 64)
				if err != nil {
					log.Warn().Err(err).Msg("Failed to parse slaves. ")
				}
				stat[fmt.Sprintf("%s_lag", key)] = lagFv
			} else {
				_, _, _, offset = kv[0], kv[1], kv[2], kv[3]
			}
			offsetKv := strings.SplitN(offset, "=", 2)
			offsetFv, err := strconv.ParseFloat(offsetKv[1], 64)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to parse slaves. ")
			}
			stat[fmt.Sprintf("%s_offset_delay", key)] = offsetFv
			continue
		}

		if re, _ := regexp.MatchString("^db", key); re {
			kv := strings.SplitN(value, ",", 3)
			keys, expires := kv[0], kv[1]

			keysKv := strings.SplitN(keys, "=", 2)
			keysFv, err := strconv.ParseFloat(keysKv[1], 64)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to parse db keys. %s")
			}
			keysStat += keysFv

			expiresKv := strings.SplitN(expires, "=", 2)
			expiresFv, err := strconv.ParseFloat(expiresKv[1], 64)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to parse db expires. %s")
			}
			expiresStat += expiresFv

			continue
		}

		stat[key], err = strconv.ParseFloat(value, 64)
		if err != nil {
			continue
		}
	}

	stat["keys"] = keysStat
	stat["expires"] = expiresStat

	if _, ok := stat["expired_keys"]; ok {
		stat["expired"] = stat["expired_keys"]
	} else {
		stat["expired"] = 0.0
	}

	for _, slave := range slaves {
		stat[fmt.Sprintf("%s_offset_delay", slave)] = stat["master_repl_offset"].(float64) - stat[fmt.Sprintf("%s_offset_delay", slave)].(float64)
	}

	return stat, nil
}

func (r RedisPlugin) GraphDefinition() map[string]plugin.Graphs {
	key := strings.Title(r.Key)
	var graphdef = map[string]plugin.Graphs{
		"queries": {
			Label: key + " Queries",
			Unit:  "integer",
			Metrics: []plugin.Metrics{
				{Name: "total_commands_processed", Label: "Queries", Diff: true},
			},
		},
		"connections": {
			Label: key + " Connections",
			Unit:  "integer",
			Metrics: []plugin.Metrics{
				{Name: "total_connections_received", Label: "Connections", Diff: true, Stacked: true},
				{Name: "rejected_connections", Label: "Rejected Connections", Diff: true, Stacked: true},
			},
		},
		"clients": {
			Label: key + " Clients",
			Unit:  "integer",
			Metrics: []plugin.Metrics{
				{Name: "connected_clients", Label: "Connected Clients", Diff: false, Stacked: true},
				{Name: "blocked_clients", Label: "Blocked Clients", Diff: false, Stacked: true},
				{Name: "connected_slaves", Label: "Connected Slaves", Diff: false, Stacked: true},
			},
		},
		"keys": {
			Label: key + " Keys",
			Unit:  "integer",
			Metrics: []plugin.Metrics{
				{Name: "keys", Label: "Keys", Diff: false},
				{Name: "expires", Label: "Keys with expiration", Diff: false},
				{Name: "expired", Label: "Expired Keys", Diff: true},
				{Name: "evicted_keys", Label: "Evicted Keys", Diff: true},
			},
		},
		"keyspace": {
			Label: key + " Keyspace",
			Unit:  "integer",
			Metrics: []plugin.Metrics{
				{Name: "keyspace_hits", Label: "Keyspace Hits", Diff: true},
				{Name: "keyspace_misses", Label: "Keyspace Missed", Diff: true},
			},
		},
		"memory": {
			Label: key + " Memory",
			Unit:  "integer",
			Metrics: []plugin.Metrics{
				{Name: "used_memory", Label: "Used Memory", Diff: false},
				{Name: "used_memory_rss", Label: "Used Memory RSS", Diff: false},
				{Name: "used_memory_peak", Label: "Used Memory Peak", Diff: false},
				{Name: "used_memory_lua", Label: "Used Memory Lua engine", Diff: false},
			},
		},
		"capacity": {
			Label: key + " Capacity",
			Unit:  "percentage",
			Metrics: []plugin.Metrics{
				{Name: "percentage_of_memory", Label: "Percentage of memory", Diff: false},
				{Name: "percentage_of_clients", Label: "Percentage of clients", Diff: false},
			},
		},
		"uptime": {
			Label: key + " Uptime",
			Unit:  "integer",
			Metrics: []plugin.Metrics{
				{Name: "uptime_in_seconds", Label: "Uptime In Seconds", Diff: false},
			},
		},
	}

	network := "tcp"
	target := fmt.Sprintf("%s:%s", r.Host, r.Port)
	if r.Socket != "" {
		target = r.Socket
		network = "unix"
	}

	c, err := redis.DialTimeout(network, target, time.Duration(5)*time.Second)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect redis.")
		return nil
	}
	defer c.Close()

	if r.Password != "" {
		if err = authenticateByPassword(c, r.Password); err != nil {
			return nil
		}
	}

	rds := c.Cmd("info")
	if rds.Err != nil {
		log.Error().Err(err).Msg("Failed to run info command.")
		return nil
	}
	str, err := rds.Str()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch information. ")
		return nil
	}

	var metricsLag []plugin.Metrics
	var metricsOffsetDelay []plugin.Metrics
	for _, line := range strings.Split(str, "\r\n") {
		if line == "" {
			continue
		}

		record := strings.SplitN(line, ":", 2)
		if len(record) < 2 {
			continue
		}
		key, _ := record[0], record[1]

		if re, _ := regexp.MatchString("^slave\\d+", key); re {
			metricsLag = append(metricsLag, plugin.Metrics{Name: fmt.Sprintf("%s_lag", key), Label: fmt.Sprintf("Replication lag to %s", key), Diff: false})
			metricsOffsetDelay = append(metricsOffsetDelay, plugin.Metrics{Name: fmt.Sprintf("%s_offset_delay", key), Label: fmt.Sprintf("Offset delay to %s", key), Diff: false})
		}
	}

	if len(metricsLag) > 0 {
		graphdef["lag"] = plugin.Graphs{
			Label:   key + " Slave Lag",
			Unit:    "seconds",
			Metrics: metricsLag,
		}
	}
	if len(metricsOffsetDelay) > 0 {
		graphdef["offset_delay"] = plugin.Graphs{
			Label:   key + " Slave Offset Delay",
			Unit:    "count",
			Metrics: metricsOffsetDelay,
		}
	}

	return graphdef
}

var (
	Revision  = "untracked"
	Version   = "0.0.0"
	GOARCH    = runtime.GOARCH
	GOOS      = runtime.GOOS
	GOVersion = runtime.Version()
)

func (r RedisPlugin) Meta() plugin.Meta {
	if r.Key == "" {
		r.Key = "redis"
	}
	version, _ := plugin.ParseVersion(Version)
	return plugin.Meta{
		Key:       r.Key,
		Type:      plugin.TypeMetrics,
		Version:   version,
		Revision:  Revision,
		GOARCH:    GOARCH,
		GOOS:      GOOS,
		GOVersion: GOVersion,
	}
}

// Do the plugin
func Do() {
	optHost := flag.String("host", "localhost", "Hostname")
	optPort := flag.String("port", "6379", "Port")
	optPassword := flag.String("password", os.Getenv("REDIS_PASSWORD"), "Password")
	optTempFile := flag.String("tempFile", "", "Temp file name")

	optSocket := flag.String("socket", "", "Server socket (overrides host and port)")
	v := flag.Bool("v", false, "version")
	flag.Parse()
	redis := RedisPlugin{}
	if *optSocket != "" {
		redis.Socket = *optSocket
	} else {
		redis.Host = *optHost
		redis.Port = *optPort
		redis.Password = *optPassword
	}
	helper := plugin.NewIdpcPlugin(redis)
	if *v {
		fmt.Println(helper.Version())
		return
	}
	if *optTempFile != "" {
		helper.TempFile = *optTempFile
	}
	helper.Run()
}
