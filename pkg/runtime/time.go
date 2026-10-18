package runtime

import (
	"fmt"
	"strings"
	"time"
)

// TimeNowUnix devolve o timestamp Unix atual em segundos.
func TimeNowUnix() int64 {
	return time.Now().Unix()
}

// TimeNowMs devolve o timestamp Unix atual em milissegundos.
func TimeNowMs() int64 {
	return time.Now().UnixMilli()
}

// TimeNowISO devolve a data/hora atual no formato ISO 8601 / RFC 3339 UTC.
func TimeNowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// TimeFormatISO converte um timestamp Unix (segundos) em string ISO 8601 UTC.
func TimeFormatISO(unixSec int64) string {
	return time.Unix(unixSec, 0).UTC().Format(time.RFC3339)
}

// TimeParseISO interpreta strings nos formatos ISO 8601 / RFC 3339 e datas simples.
func TimeParseISO(s string) Result[int64, string] {
	s = strings.TrimSpace(s)
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			return Ok[int64, string](t.Unix())
		}
	}
	return Err[int64, string](fmt.Sprintf("formato de data/hora invalido: %q", s))
}

// TimeFormat formata um timestamp Unix. Converte automaticamente mascaras comuns
// amigaveis (ex: YYYY-MM-DD, DD/MM/YYYY HH:mm:ss) para o padrao de formatacao Go.
func TimeFormat(unixSec int64, layout string) string {
	t := time.Unix(unixSec, 0).UTC()
	if layout == "" || strings.EqualFold(layout, "iso") || strings.EqualFold(layout, "rfc3339") {
		return t.Format(time.RFC3339)
	}

	goLayout := convertToGolangLayout(layout)
	return t.Format(goLayout)
}

// convertToGolangLayout traduz mascaras ISO/humanas para o padrao Go (2006-01-02 15:04:05).
func convertToGolangLayout(layout string) string {
	replacer := strings.NewReplacer(
		"YYYY", "2006",
		"yyyy", "2006",
		"MM", "01",
		"DD", "02",
		"dd", "02",
		"HH", "15",
		"hh", "03",
		"mm", "04",
		"ss", "05",
		"SS", "05",
	)
	return replacer.Replace(layout)
}

// TimeAddSeconds soma (ou subtrai) segundos de um timestamp.
func TimeAddSeconds(unixSec int64, seconds int64) int64 {
	return unixSec + seconds
}

// TimeDiffSeconds calcula a diferenca (a - b) em segundos entre dois timestamps.
func TimeDiffSeconds(unixSecA int64, unixSecB int64) int64 {
	return unixSecA - unixSecB
}

// TimeYear devolve o ano (ex: 2026).
func TimeYear(unixSec int64) int64 {
	return int64(time.Unix(unixSec, 0).UTC().Year())
}

// TimeMonth devolve o mes (1-12).
func TimeMonth(unixSec int64) int64 {
	return int64(time.Unix(unixSec, 0).UTC().Month())
}

// TimeDay devolve o dia do mes (1-31).
func TimeDay(unixSec int64) int64 {
	return int64(time.Unix(unixSec, 0).UTC().Day())
}

// TimeHour devolve a hora do dia (0-23).
func TimeHour(unixSec int64) int64 {
	return int64(time.Unix(unixSec, 0).UTC().Hour())
}

// TimeMinute devolve o minuto da hora (0-59).
func TimeMinute(unixSec int64) int64 {
	return int64(time.Unix(unixSec, 0).UTC().Minute())
}

// TimeSecond devolve o segundo do minuto (0-59).
func TimeSecond(unixSec int64) int64 {
	return int64(time.Unix(unixSec, 0).UTC().Second())
}

// TimeWeekday devolve o dia da semana (0 = Domingo, 1 = Segunda, ..., 6 = Sabado).
func TimeWeekday(unixSec int64) int64 {
	return int64(time.Unix(unixSec, 0).UTC().Weekday())
}
