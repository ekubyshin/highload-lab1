package dashboard

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	prometheusJob  = "items-app"
	metricRequests = "http_requests_total"
	metricDuration = "http_request_duration_seconds_bucket"
)

// methodPattern matches the method tokens the application registrar accepts.
var methodPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,15}$`)

// promqlString quotes value as a PromQL string literal.
// JSON encoding of the surrounding document is a separate step.
func promqlString(value string) string {
	var b strings.Builder
	b.Grow(len(value) + 2)
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func routeSelector(method, route string) string {
	return strings.Join([]string{
		"job=" + promqlString(prometheusJob),
		"method=" + promqlString(method),
		"route=" + promqlString(route),
		`datacenter=~"${datacenter:regex}"`,
		`pod_id=~"${pod_id:regex}"`,
	}, ",")
}

func latencyQuery(method, route string, quantile float64) string {
	return fmt.Sprintf(
		"histogram_quantile(%s, sum by (le) (rate(%s{%s}[$__rate_interval])))",
		strconv.FormatFloat(quantile, 'f', -1, 64),
		metricDuration,
		routeSelector(method, route),
	)
}

func rpsQuery(method, route string) string {
	return fmt.Sprintf(
		"sum(rate(%s{%s}[$__rate_interval]))",
		metricRequests,
		routeSelector(method, route),
	)
}

func errorQuery(method, route string) string {
	selector := routeSelector(method, route) + `,status_code=~"4..|5.."`
	return fmt.Sprintf(
		"sum by (status_code) (rate(%s{%s}[$__rate_interval]))",
		metricRequests,
		selector,
	)
}

func datacenterVariableQuery() string {
	return fmt.Sprintf(
		"label_values(%s{job=%s}, datacenter)",
		metricRequests,
		promqlString(prometheusJob),
	)
}

func podVariableQuery() string {
	return fmt.Sprintf(
		"label_values(%s{job=%s,datacenter=~\"${datacenter:regex}\"}, pod_id)",
		metricRequests,
		promqlString(prometheusJob),
	)
}
