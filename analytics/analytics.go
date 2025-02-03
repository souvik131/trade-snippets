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

const (
	prefix    = "market"
	batchSize = 10000
)

var (
	conn                 driver.Conn
	DEFAULT_DATA_TYPE_CH = Fields{
		"float32":   "Int32 DEFAULT -1 CODEC(Delta, ZSTD(3))",  // Stores as value * 100
		"float64":   "Int64 DEFAULT -1 CODEC(Delta, ZSTD(3))",  // Stores as value * 100
		"ufloat32":  "UInt32 DEFAULT -1 CODEC(Delta, ZSTD(3))", // Stores as value * 100
		"ufloat64":  "UInt64 DEFAULT -1 CODEC(Delta, ZSTD(3))", // Stores as value * 100
		"int32":     "Int32 DEFAULT -1 CODEC(Delta, ZSTD(3))",
		"int64":     "Int64 DEFAULT -1 CODEC(Delta, ZSTD(3))",
		"uint32":    "UInt32 DEFAULT -1 CODEC(Delta, ZSTD(3))",
		"uint64":    "UInt64 DEFAULT -1 CODEC(Delta, ZSTD(3))",
		"string":    "LowCardinality(String)",
		"timestamp": "DateTime64(3) CODEC(Delta, ZSTD(3))",
	}

	LIVE_DATA       LogType = "LIVE_DATA"
	DERIVED_OPTIONS LogType = "DERIVED_OPTIONS"
	GREEKS          LogType = "GREEKS"

	schemas = Schema{
		LIVE_DATA: {
			"exchange":              "string",
			"script":                "string",
			"expiry":                "string",
			"strike":                "string",
			"instrument_type":       "string",
			"timestamp":             "timestamp",
			"last_price":            "ufloat32",
			"lot_size":              "ufloat32",
			"open":                  "ufloat32",
			"high":                  "ufloat32",
			"low":                   "ufloat32",
			"close":                 "ufloat32",
			"volume":                "uint32",
			"oi":                    "uint32",
			"last_traded_timestamp": "timestamp",

			"buy_price_1":     "ufloat32",
			"buy_quantity_1":  "ufloat32",
			"buy_order_1":     "ufloat32",
			"sell_price_1":    "ufloat32",
			"sell_quantity_1": "ufloat32",
			"sell_order_1":    "ufloat32",

			"buy_price_2":     "ufloat32",
			"buy_quantity_2":  "ufloat32",
			"buy_order_2":     "ufloat32",
			"sell_price_2":    "ufloat32",
			"sell_quantity_2": "ufloat32",
			"sell_order_2":    "ufloat32",

			"buy_price_3":     "ufloat32",
			"buy_quantity_3":  "ufloat32",
			"buy_order_3":     "ufloat32",
			"sell_price_3":    "ufloat32",
			"sell_quantity_3": "ufloat32",
			"sell_order_3":    "ufloat32",

			"buy_price_4":     "ufloat32",
			"buy_quantity_4":  "ufloat32",
			"buy_order_4":     "ufloat32",
			"sell_price_4":    "ufloat32",
			"sell_quantity_4": "ufloat32",
			"sell_order_4":    "ufloat32",

			"buy_price_5":     "ufloat32",
			"buy_quantity_5":  "ufloat32",
			"buy_order_5":     "ufloat32",
			"sell_price_5":    "ufloat32",
			"sell_quantity_5": "ufloat32",
			"sell_order_5":    "ufloat32",
		},
		GREEKS: {
			"exchange":        "string",
			"script":          "string",
			"expiry":          "string",
			"strike":          "string",
			"instrument_type": "string",
			"timestamp":       "timestamp",
			"delta":           "float32",
			"gamma":           "float32",
			"vega":            "float32",
			"theta":           "float32",
			"rho":             "float32",
		},
		DERIVED_OPTIONS: {
			"script":         "string",
			"expiry":         "string",
			"timestamp":      "timestamp",
			"atm_strike":     "ufloat32",
			"atm_iv":         "ufloat32",
			"straddle_price": "ufloat32",
		},
	}
	mu sync.Mutex
)

func Init() {
	dbURI := os.Getenv("DB_URI")
	if dbURI == "" {
		log.Panic("DB_URI environment variable is not set")
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		log.Panic("DB_NAME environment variable is not set")
	}

	var (
		con, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{dbURI},
			Auth: clickhouse.Auth{
				Database: dbName,
			},
			Settings: clickhouse.Settings{
				"max_execution_time": 60,
			},
			Compression: &clickhouse.Compression{
				Method: clickhouse.CompressionLZ4,
			},
			DialTimeout:          time.Second * 30,
			MaxOpenConns:         5,
			MaxIdleConns:         5,
			ConnMaxLifetime:      time.Hour,
			ConnOpenStrategy:     clickhouse.ConnOpenInOrder,
			BlockBufferSize:      10,
			MaxCompressionBuffer: 10240,
		})
	)
	if err != nil {
		log.Panicf("Failed to create database connection: %v", err)
	}

	// Test the connection
	if err := con.Ping(context.Background()); err != nil {
		log.Panicf("Failed to ping database: %v", err)
	}

	mu.Lock()
	conn = con
	mu.Unlock()

	log.Info("Successfully connected to ClickHouse database")

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
		if typeOf == "string" || typeOf == "timestamp" {
			primaryKeyString += name + ", "
		}
	}

	query := `CREATE TABLE IF NOT EXISTS ` + dbName + `(` + strings.Join(dataArr, ", ") + `) ENGINE = ReplacingMergeTree  PRIMARY KEY (` + primaryKeyString + ` ) ORDER BY (` + primaryKeyString + ` )`

	mu.Lock()
	_, err := conn.Query(ctx, query)
	mu.Unlock()
	if err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
			log.Panic(err)
		}
	}
}

func (l LogType) BatchStore(fields []*log.Fields) {
	if conn == nil {
		log.Panic("analytics_not_initiated")
	}
	if len(fields) == 0 {
		return
	}

	ctx := context.Background()
	dbName := fmt.Sprintf("%v_%v_log", prefix, strings.ToLower(string(l)))
	var keyArr []string

	// Get keys from first record to ensure consistent column order
	for key := range *fields[0] {
		if _, ok := schemas[l][key]; ok {
			keyArr = append(keyArr, key)
		}
	}

	// Process records in batches
	for i := 0; i < len(fields); i += batchSize {
		end := i + batchSize
		if end > len(fields) {
			end = len(fields)
		}

		log.Infof("Saving %v records to %v, from index %v to %v\n", end-i, dbName, i, end)
		mu.Lock()
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO "+dbName+" ("+strings.Join(keyArr, ", ")+")")
		if err != nil {
			mu.Unlock()
			log.Panicf("Failed to prepare batch: %v", err)
		}

		// Process each record in the current batch
		for _, f := range fields[i:end] {
			values := make([]interface{}, 0, len(keyArr))
			for _, key := range keyArr {
				if val, ok := (*f)[key]; ok {
					if _, ok := schemas[l][key]; ok {

						values = append(values, val)

					}
				} else {
					// Handle missing field with default value
					values = append(values, -1)
				}
			}

			if err := batch.Append(values...); err != nil {
				mu.Unlock()
				log.Panicf("Failed to append to batch: %v", err)
			}
		}

		if err := batch.Send(); err != nil {
			mu.Unlock()
			log.Panicf("Failed to send batch: %v", err)
		}
		mu.Unlock()
	}
}
