#!/usr/bin/env bash
# test_all_params.sh – runs PSFuzz with various parameter combinations to verify
# flags and modules work. Each run is short (-maxtime). Requires network access.
# Run from project root (or set PSFUZZ_PROJECT_ROOT); config.example.json is resolved relative to project root.
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="${PSFUZZ_PROJECT_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"
cd "$PROJECT_ROOT"

PSFUZZ_BIN="${PSFUZZ_BIN:-./psfuzz}"
BASE_URL="${PSFUZZ_BASE_URL:-https://example.com}"
OUT_DIR="${PSFUZZ_PARAMTEST_OUT:-./paramtest_out}"
WORDLIST="${PSFUZZ_WORDLIST:-fav}"
MAXTIME_SHORT=5
MAXTIME_MED=8
CONFIG_EXAMPLE="$PROJECT_ROOT/config.example.json"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

run_test() {
	local name="$1"
	shift
	local out="$OUT_DIR/$name"
	mkdir -p "$OUT_DIR"
	if "$PSFUZZ_BIN" -u "$BASE_URL/FUZZ" -w "$WORDLIST" -o "$out" -maxtime "$MAXTIME_SHORT" -q "$@" 2>/dev/null; then
		echo -e "${GREEN}PASS${NC} $name"
		return 0
	else
		echo -e "${RED}FAIL${NC} $name"
		return 1
	fi
}

run_test_maxtime() {
	local name="$1"
	local mt="$2"
	shift 2
	local out="$OUT_DIR/$name"
	mkdir -p "$OUT_DIR"
	if "$PSFUZZ_BIN" -u "$BASE_URL/FUZZ" -w "$WORDLIST" -o "$out" -maxtime "$mt" -q "$@" 2>/dev/null; then
		echo -e "${GREEN}PASS${NC} $name"
		return 0
	else
		echo -e "${RED}FAIL${NC} $name"
		return 1
	fi
}

echo "PSFuzz parameter test script"
echo "  Binary: $PSFUZZ_BIN"
echo "  URL:    $BASE_URL"
echo "  Out:    $OUT_DIR"
echo ""

FAIL=0

# --- Basic ---
run_test "01_basic" || ((FAIL++))
run_test "02_concurrency" -c 3 || ((FAIL++))

# --- Rate & delay ---
run_test "03_rate" -rate 20 || ((FAIL++))
run_test "04_delay" -p 0.05 || ((FAIL++))

# --- Headers ---
run_test "05_headers" -H "Accept: text/html" -H "X-PSFuzz-Test: 1" || ((FAIL++))

# --- Response size ---
run_test "06_max_size" -max-size 100000 || ((FAIL++))

# --- Filters: status ---
run_test "07_mc" -mc 200,301,404 || ((FAIL++))
run_test "08_fc" -fc 500 || ((FAIL++))

# --- Filters: length/words/lines ---
run_test "09_fl" -fl 0-10000 || ((FAIL++))
run_test "10_mw" -mw 0-1000 || ((FAIL++))

# --- Output formats ---
run_test_maxtime "11_of_txt" 3 -of txt || ((FAIL++))
run_test_maxtime "12_of_json" 3 -of json || ((FAIL++))
run_test_maxtime "13_of_csv" 3 -of csv || ((FAIL++))
run_test_maxtime "14_of_ndjson" 3 -of ndjson || ((FAIL++))
run_test_maxtime "15_of_html" 3 -of html || ((FAIL++))

# --- Modules ---
run_test "16_modules" -modules fingerprint,cors,urlextract,links -of json || ((FAIL++))

# --- Audit log ---
run_test "17_audit" -audit-log "$OUT_DIR/audit.ndjson" -audit-max-body 5000 || ((FAIL++))
if [[ -f "$OUT_DIR/audit.ndjson" ]]; then
	lines=$(wc -l < "$OUT_DIR/audit.ndjson")
	echo "      audit.ndjson: $lines lines"
fi

# --- Extracted URLs file ---
run_test "18_extracted_urls" -modules links -extracted-urls-file "$OUT_DIR/extracted.txt" -of json || ((FAIL++))
if [[ -f "$OUT_DIR/extracted.txt" ]]; then
	lines=$(wc -l < "$OUT_DIR/extracted.txt")
	echo "      extracted.txt: $lines URLs"
fi

# --- Enqueue module URLs + depth (short) ---
run_test_maxtime "19_enqueue_depth" "$MAXTIME_MED" -modules links -enqueue-module-urls links -D 1 -of json || ((FAIL++))

# --- Extensions ---
run_test_maxtime "20_extensions" 3 -e php,txt || ((FAIL++))

# --- Verbs ---
run_test_maxtime "21_verbs" 3 -verbs GET,HEAD || ((FAIL++))

# --- Stop conditions ---
run_test_maxtime "22_stop_after" 10 -sa 3 || ((FAIL++))

# --- Bypass (budget 1) ---
run_test "23_bypass" -b -bypass-budget 1 || ((FAIL++))

# --- WAF adaptive ---
run_test "24_waf" -waf-adaptive -waf-threshold 5 -waf-factor 1.5 || ((FAIL++))

# --- TLS skip ---
run_test "25_insecure" -insecure || ((FAIL++))

# --- Config file (if example exists) ---
if [[ -f "$CONFIG_EXAMPLE" ]]; then
	run_test_maxtime "26_config_file" 3 -cf "$CONFIG_EXAMPLE" -u "$BASE_URL/FUZZ" -w "$WORDLIST" || ((FAIL++))
else
	echo "  skip 26_config_file (no config.example.json at $CONFIG_EXAMPLE)"
fi

# --- Save config ---
run_test_maxtime "27_save_config" 2 -save-config "$OUT_DIR/saved.json" || ((FAIL++))
if [[ -f "$OUT_DIR/saved.json" ]]; then
	echo "      saved.json: $(wc -c < "$OUT_DIR/saved.json") bytes"
fi

# --- Presets ---
run_test_maxtime "28_preset_quick" 3 -preset quick || ((FAIL++))

# --- Version / help (no network) ---
if "$PSFUZZ_BIN" -v 2>/dev/null | grep -q PSFuzz; then
	echo -e "${GREEN}PASS${NC} 29_version"
else
	echo -e "${RED}FAIL${NC} 29_version"
	((FAIL++))
fi
if "$PSFUZZ_BIN" -h 2>/dev/null | grep -q "USAGE"; then
	echo -e "${GREEN}PASS${NC} 30_help"
else
	echo -e "${RED}FAIL${NC} 30_help"
	((FAIL++))
fi

echo ""
if [[ $FAIL -eq 0 ]]; then
	echo -e "${GREEN}All parameter tests passed.${NC}"
	exit 0
else
	echo -e "${RED}$FAIL test(s) failed.${NC}"
	exit 1
fi
