package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func loadCsv(filename string) [][]string {
	f, err := os.Open(filename)
	panicIf(err)
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	records, err := r.ReadAll()
	panicIf(err)
	return records
}

func loadToken(filename string) [][]string {
	records := loadCsv(filename)
	if len(records) < 2 {
		panicIf(fmt.Errorf("filename %s error", filename))
	}
	return records[1:]
}

func loadTemplate(filename string) (records [][]string, headers []string, H map[string]int) {
	r := loadCsv(filename)
	headers = r[1]
	for idx, header := range headers {
		headers[idx] = strip(header)
	}
	records = r[2:]
	H = make(map[string]int, len(headers))
	for idx, v := range headers {
		H[v] = idx
	}
	return
}

func strip(s string) string {
	return strings.Trim(s, " ")
}

func stringToList(s string) []string {
	s = strip(s)
	s = strings.ReplaceAll(s, "，", ",")
	s = strip(s)
	ret := strings.Split(s, ",")
	for idx, value := range ret {
		ret[idx] = strip(value)
	}
	return ret
}

func listToString(sl []string) string {
	return strings.Join(sl, ", ")
}

func stringToBool(s string) bool {
	enable := strip(s)
	if enable == "是" {
		return true
	} else {
		return false
	}
}

func boolToString(b bool) string {
	if b {
		return "是"
	}
	return "否"
}

func stringToInt(s string) int {
	s = strip(s)
	ret, err := strconv.Atoi(s)
	if err != nil {
		fmt.Printf("%v: stringToInt(%s)\n", err, s)
	}
	return ret
}

func intToString(i int) string {
	return strconv.Itoa(i)
}

func intListToString(isl []int) string {
	sl := make([]string, 0)
	for _, i := range isl {
		sl = append(sl, strconv.Itoa(i))
	}
	return strings.Join(sl, ", ")
}

func formatTime(t string) string {
	i, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		fmt.Printf("%v: formatTime(%s)\n", err, t)
	}
	tt := time.Unix(i, 0)
	return tt.Format("2006-01-02|15:04:05")
}

func progressBar(lengthCh, indexCh <-chan int, stopCh <-chan struct{}) {
	var buf [50]byte
	for {
		select {
		case length := <-lengthCh:
			index := <-indexCh
			progress := float32(index+1) / float32(length) * 100
			p := int(progress) / 2
			for i := 0; i < p; i++ {
				buf[i] = '='
			}
			if p < 50 {
				buf[p] = '>'
			}
			for i := p + 1; i < 50; i++ {
				buf[i] = ' '
			}
			fmt.Printf("\r\033[K%.2f%% [%s] %d/%d ", progress, buf, index+1, length)
		case <-stopCh:
			return
		}
	}
}
