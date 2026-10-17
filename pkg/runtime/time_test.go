package runtime

import (
	"testing"
)

func TestTimeNowAndBasics(t *testing.T) {
	now := TimeNowUnix()
	if now < 1700000000 {
		t.Fatalf("timestamp unix invalido: %d", now)
	}

	ms := TimeNowMs()
	if ms < now*1000 {
		t.Fatalf("timestamp ms invalido: %d vs %d", ms, now)
	}

	iso := TimeNowISO()
	if len(iso) < 19 {
		t.Fatalf("iso invalido: %s", iso)
	}

	parsed := TimeParseISO(iso)
	if !parsed.OK {
		t.Fatalf("falha ao interpretar iso gerado: %s", parsed.Error)
	}
	if parsed.Value != now && parsed.Value != now-1 && parsed.Value != now+1 {
		t.Fatalf("diferenca no parse de now: %d vs %d", parsed.Value, now)
	}
}

func TestTimeParsingAndFormatting(t *testing.T) {
	// 2026-10-17 12:00:00 UTC = 1792238400
	isoStr := "2026-10-17T12:00:00Z"
	res := TimeParseISO(isoStr)
	if !res.OK {
		t.Fatalf("erro no parse de %s: %s", isoStr, res.Error)
	}
	ts := res.Value

	formattedISO := TimeFormatISO(ts)
	if formattedISO != isoStr {
		t.Fatalf("esperado %s, obteve %s", isoStr, formattedISO)
	}

	formattedCustom := TimeFormat(ts, "DD/MM/YYYY HH:mm:ss")
	if formattedCustom != "17/10/2026 12:00:00" {
		t.Fatalf("esperado '17/10/2026 12:00:00', obteve %q", formattedCustom)
	}

	dateOnly := TimeFormat(ts, "YYYY-MM-DD")
	if dateOnly != "2026-10-17" {
		t.Fatalf("esperado '2026-10-17', obteve %q", dateOnly)
	}

	// Parsing de data simples sem hora
	resDate := TimeParseISO("2026-10-17")
	if !resDate.OK {
		t.Fatalf("erro no parse de data simples: %s", resDate.Error)
	}
}

func TestTimeOperations(t *testing.T) {
	ts := int64(1792238400) // 2026-10-17 12:00:00 UTC (Sabado = 6)

	if y := TimeYear(ts); y != 2026 {
		t.Fatalf("ano esperado 2026, obteve %d", y)
	}
	if m := TimeMonth(ts); m != 10 {
		t.Fatalf("mes esperado 10, obteve %d", m)
	}
	if d := TimeDay(ts); d != 17 {
		t.Fatalf("dia esperado 17, obteve %d", d)
	}
	if h := TimeHour(ts); h != 12 {
		t.Fatalf("hora esperada 12, obteve %d", h)
	}
	if w := TimeWeekday(ts); w != 6 {
		t.Fatalf("dia da semana esperado 6 (Sabado), obteve %d", w)
	}

	added := TimeAddSeconds(ts, 3600)
	if diff := TimeDiffSeconds(added, ts); diff != 3600 {
		t.Fatalf("diferenca esperada 3600, obteve %d", diff)
	}
}
