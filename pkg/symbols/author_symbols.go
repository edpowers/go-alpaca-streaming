package author_symbols

import (
	"log"
    "crypto/tls"
	"encoding/json"
	"net/http"
	"github.com/xitongsys/parquet-go-source/local"
    "github.com/xitongsys/parquet-go/reader"
)


type Symbol struct {
    Symbol string `parquet:"name=symbol, type=UTF8"`
}

const maxLocalSymbols = 500  // Constant to define max number of symbols


func GetAuthorSymbols() ([]string, error) {
    var symbols []string

    // Create a custom HTTP client
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{Transport: tr}


    // Perform the HTTP request
    response, err := client.Get("https://algotrading.ventures/datastreaming/v1/datasets/author_symbols")
    if err != nil {
        return nil, err
    }
    defer response.Body.Close()

    err = json.NewDecoder(response.Body).Decode(&symbols)
    if err != nil {
        return nil, err
    }

    return symbols, nil
}


// Fallback to fetch symbols from local Parquet file
func GetLocalSymbols() []string {
    localFilePath := "/data/deriv_symbols_used.parquet"
    fr, err := local.NewLocalFileReader(localFilePath)
    if err != nil {
        log.Fatal("Can't open local file: ", err)
        return nil
    }

    pr, err := reader.NewParquetReader(fr, new(Symbol), 4)
    if err != nil {
        log.Fatal("Can't create parquet reader: ", err)
        return nil
    }

    var symbols []string
    symbolsBuffer := make([]Symbol, 1)

    numRows := int(pr.GetNumRows())
    if numRows > maxLocalSymbols {
        numRows = maxLocalSymbols  // Limit to 500 symbols
    }

    for i := 0; i < numRows; i++ {
        if err := pr.Read(&symbolsBuffer); err != nil {
            log.Println("Read error: ", err)
        }
        symbols = append(symbols, symbolsBuffer[0].Symbol)
    }

    pr.ReadStop()
    fr.Close()
    return symbols
}