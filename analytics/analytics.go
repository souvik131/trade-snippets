package analytics

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	log "github.com/sirupsen/logrus"
)

type Fields map[string]string
type Schema map[LogType]log.Fields
type LogType string

const prefix string = "market"

var (
	conn                 driver.Conn
	mut                  = &sync.Mutex{}
	DEFAULT_DATA_TYPE_CH = Fields{
		"float":       "UInt32 DEFAULT -1 CODEC(Delta, ZSTD(3))", // Stores as value * 100
		"int":         "Int32 DEFAULT -1 CODEC(Delta, ZSTD(3))",
		"uint":        "UInt64 DEFAULT -1 CODEC(Delta, ZSTD(3))",
		"string":      "LowCardinality(String)",
		"array_float": "Array(UInt32) CODEC (Delta, ZSTD(3))",
		"array_int":   "Array(UInt32) CODEC (Delta, ZSTD(3))",
	}

	MARKET_DATA LogType = "MARKET_DATA"
	INDICATORS  LogType = "INDICATORS"
	STATS_ARB   LogType = "STATS_ARB"
	VOL_SKEW    LogType = "VOL_SKEW"
	OPTIONS     LogType = "OPTIONS"

	schemas = Schema{
		MARKET_DATA: {
			"tradingsymbol":         "string",
			"exchange":              "string",
			"last_price":            "float",
			"open":                  "float",
			"high":                  "float",
			"low":                   "float",
			"close":                 "float",
			"volume":                "int",
			"oi":                    "int",
			"last_traded_timestamp": "int",
			"buy_price":             "array_float",
			"buy_quantity":          "array_int",
			"sell_price":            "array_float",
			"sell_quantity":         "array_int",
		},
		INDICATORS: {
			"tradingsymbol":   "string",
			"ema_5":           "float",
			"ema_10":          "float",
			"ema_50":          "float",
			"rsi_14":          "float",
			"macd":            "float",
			"bollinger_upper": "float",
			"bollinger_lower": "float",
			"vwap":            "float",
			"z_score":         "float",
			"vwap_spread":     "float",
		},
		STATS_ARB: {
			"pair_1":         "string",
			"pair_2":         "string",
			"cointegration":  "float",
			"spread":         "float",
			"z_score_spread": "float",
			"hedge_ratio":    "float",
		},
		VOL_SKEW: {
			"script":                  "string",
			"realized_vol":            "float",
			"implied_vol":             "float",
			"volatility_risk_premium": "float",
			"skew_ratio":              "float",
			"risk_reversal":           "float",
			"term_structure_slope":    "float",
		},
		OPTIONS: {
			"tradingsymbol": "string",
			"delta":         "float",
			"gamma":         "float",
			"vega":          "float",
			"theta":         "float",
			"rho":           "float",
		},
	}
)

func Init() {
	var (
		con, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{os.Getenv("DB_URI")},
			Auth: clickhouse.Auth{
				Database: os.Getenv("DB_NAME"),
			},
		})
	)
	if err != nil {
		log.Panic(err)
	}
	mut.Lock()
	conn = con
	mut.Unlock()

	for name, f := range schemas {
		dbName := fmt.Sprintf("%v_%v_log", prefix, strings.ToLower(string(name)))
		CreateTables(dbName, f)
	}
}

func CreateTables(dbName string, f log.Fields) {
	if conn == nil {
		log.Panic("no_connection")
	}
	ctx := context.Background()
	dataArr := make([]string, 0)
	primaryKeyString := ""

	for name, typeOf := range f {
		typeOfString := fmt.Sprintf("%v", typeOf)
		if chType, ok := DEFAULT_DATA_TYPE_CH[typeOfString]; ok {
			dataArr = append(dataArr, name+" "+chType)
		} else {
			log.Panic("data_type_not_valid")
		}
		if typeOf == "string" {
			primaryKeyString += name + ", "
		}
	}

	query := `CREATE TABLE IF NOT EXISTS ` + dbName + `(
		timestamp DateTime64(3) CODEC(Delta, ZSTD(3)), ` + strings.Join(dataArr, ", ") + `
	) ENGINE = ReplacingMergeTree  PRIMARY KEY (` + primaryKeyString + ` timestamp) ORDER BY (` + primaryKeyString + ` timestamp)`
	log.Println(query)

	mut.Lock()
	_, err := conn.Query(ctx, query)
	mut.Unlock()
	if err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
			log.Panic(err)
		}
	}
}

func StoreLogs(l LogType, f *log.Fields) {
	if conn == nil {
		log.Panic("analytics_not_initiated")
	}
	keyArr, valueArr := make([]string, 0), make([]string, 0)
	for key, val := range *f {
		if typeOf, ok := schemas[l][key]; ok {
			if typeOf == "int" || typeOf == "float" {
				valueArr = append(valueArr, fmt.Sprintf("%v", val))
			} else {
				valueArr = append(valueArr, fmt.Sprintf("'%v'", val))
			}
			keyArr = append(keyArr, key)
		} else {
			log.Panic("fields_do_not_match")
		}
	}
	keyArr = append(keyArr, "timestamp")
	valueArr = append(valueArr, fmt.Sprintf("'%v'", time.Now().UnixMilli()))

	dbName := fmt.Sprintf("%v_%v_log", prefix, strings.ToLower(string(l)))
	query := "INSERT INTO " + dbName + " (" + strings.Join(keyArr, ", ") + ") VALUES (" + strings.Join(valueArr, ", ") + ") ;"

	mut.Lock()
	_, err := conn.Query(context.Background(), query)
	mut.Unlock()
	if err != nil {
		log.Panic(err)
	}
}
