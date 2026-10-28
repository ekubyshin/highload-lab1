package dashboard

import (
	"fmt"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	gf "github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

func init() {
	// JSON coming back from Grafana names queries and panel options by plugin id.
	// The SDK decodes those variants only after they are registered.
	cog.NewRuntime().RegisterDataqueryVariant(prometheus.VariantConfig())
	cog.NewRuntime().RegisterPanelcfgVariant(timeseries.VariantConfig())
}

// Build creates the dashboard for handlers in catalog order.
// Routes that disappear from a later catalog are omitted, so an update replaces them.
func Build(handlers []Endpoint) (Spec, error) {
	if len(handlers) == 0 {
		return Spec{}, fmt.Errorf("handlers catalog is empty")
	}
	builder := gf.NewDashboardBuilder(dashboardTitle).
		Uid(dashboardUID).
		Description("Generated from the application handler catalog. Editing in the UI is disabled.").
		Readonly().
		Refresh("5s").
		Timezone("browser").
		Tags([]string{dashboardUID}).
		Time("now-15m", "now").
		WithVariable(datacenterVariable()).
		WithVariable(podVariable())
	for i, handler := range handlers {
		if err := handler.validate(); err != nil {
			return Spec{}, err
		}
		for _, panel := range panelsFor(handler, i*3+1, i*8) {
			builder.WithPanel(panel)
		}
	}
	spec, err := builder.Build()
	if err != nil {
		return Spec{}, err
	}
	if spec.SchemaVersion != schemaVersion {
		return Spec{}, fmt.Errorf("foundation sdk schemaVersion = %d, want %d", spec.SchemaVersion, schemaVersion)
	}
	return spec, nil
}

func panelsFor(handler Endpoint, id, y int) []*timeseries.PanelBuilder {
	base := handler.Method + " " + handler.Path
	return []*timeseries.PanelBuilder{
		newPanel(id, base+" latency", "Latency percentiles from histogram buckets summed by le. Empty ranges stay unknown.", "s", y, 0).
			WithTarget(queryTarget("A", latencyQuery(handler.Method, handler.Path, 0.5), "p50")).
			WithTarget(queryTarget("B", latencyQuery(handler.Method, handler.Path, 0.95), "p95")).
			WithTarget(queryTarget("C", latencyQuery(handler.Method, handler.Path, 0.99), "p99")),
		newPanel(id+1, base+" RPS", "Requests per second for the selected datacenters and pods.", "reqps", y, 8).
			WithTarget(queryTarget("A", rpsQuery(handler.Method, handler.Path), "rps")),
		newPanel(id+2, base+" errors", "Error responses per second, one series for each HTTP status code.", "reqps", y, 16).
			WithTarget(queryTarget("A", errorQuery(handler.Method, handler.Path), "{{status_code}}")),
	}
}

func newPanel(id int, title, description, unit string, y, x int) *timeseries.PanelBuilder {
	return timeseries.NewPanelBuilder().
		Id(uint32(id)).
		Title(title).
		Description(description).
		Datasource(prometheusRef()).
		GridPos(gf.GridPos{H: 8, W: 8, X: uint32(x), Y: uint32(y)}).
		Unit(unit).
		SpanNulls(common.BoolOrFloat64{Bool: boolPtr(false)}).
		DrawStyle(common.GraphDrawStyleLine).
		LineInterpolation(common.LineInterpolationLinear).
		ShowPoints(common.VisibilityModeNever).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode(common.LegendDisplayModeList).
			Placement(common.LegendPlacementBottom).
			ShowLegend(true).
			Calcs([]string{})).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode(common.TooltipDisplayModeMulti).
			Sort(common.SortOrderNone))
}

func queryTarget(refID, expr, legend string) *prometheus.DataqueryBuilder {
	return prometheus.NewDataqueryBuilder().
		RefId(refID).
		Datasource(prometheusRef()).
		Expr(expr).
		LegendFormat(legend).
		EditorMode(prometheus.QueryEditorModeCode).
		Range()
}

func prometheusRef() common.DataSourceRef {
	kind := "prometheus"
	uid := datasourceUID
	return common.DataSourceRef{Type: &kind, Uid: &uid}
}

func datacenterVariable() *gf.QueryVariableBuilder {
	return baseVariable("datacenter", "Datacenter", datacenterVariableQuery())
}

func podVariable() *gf.QueryVariableBuilder {
	return baseVariable("pod_id", "Pod", podVariableQuery())
}

func baseVariable(name, label, query string) *gf.QueryVariableBuilder {
	selected := true
	allText := "All"
	allValue := allVariableValue
	option := gf.VariableOption{
		Selected: &selected,
		Text:     gf.StringOrArrayOfString{String: &allText},
		Value:    gf.StringOrArrayOfString{String: &allValue},
	}
	return gf.NewQueryVariableBuilder(name).
		Label(label).
		Query(gf.StringOrMap{String: &query}).
		Definition(query).
		Datasource(prometheusRef()).
		// Refresh on load and when the time range changes.
		Refresh(gf.VariableRefreshOnTimeRangeChanged).
		Sort(gf.VariableSortAlphabeticalAsc).
		IncludeAll(true).
		AllValue(allRegexValue).
		Multi(true).
		Current(option).
		// All is the saved selection. An empty metric list must not replace it
		// with a concrete value that would hide series after traffic starts.
		Options([]gf.VariableOption{option})
}

func expressionsOf(spec Spec) []string {
	out := make([]string, 0, len(spec.Panels))
	for _, item := range spec.Panels {
		if item.Panel == nil {
			continue
		}
		for _, target := range item.Panel.Targets {
			out = append(out, targetExpr(target))
		}
	}
	return out
}

func targetExpr(target any) string {
	switch query := target.(type) {
	case prometheus.Dataquery:
		return query.Expr
	case *prometheus.Dataquery:
		return query.Expr
	default:
		return ""
	}
}

func queryText(query *gf.StringOrMap) string {
	if query == nil {
		return ""
	}
	if query.String != nil {
		return *query.String
	}
	if text, ok := query.Map["query"].(string); ok {
		return text
	}
	return ""
}

func optionString(value gf.StringOrArrayOfString) string {
	if value.String != nil {
		return *value.String
	}
	if len(value.ArrayOfString) == 1 {
		return value.ArrayOfString[0]
	}
	return ""
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolPtr(value bool) *bool {
	return &value
}

func isEditable(spec Spec) bool {
	return spec.Editable != nil && *spec.Editable
}
