#!/usr/bin/env python3

from bokeh.plotting import figure, output_file, show
from bokeh.io import export_svgs
from bokeh.models import WheelZoomTool, HoverTool, Range1d, LinearAxis, FuncTickFormatter
from bokeh.palettes import Spectral6

from datetime import datetime as dt
import json
import math

with open("../usage.json") as f:
	data = json.load(f)

categories = ["transaction", "transaction-proposal", "endorsement", "non-revocation-request", "non-revocation-handle", "credentials"]
sourceData = {
	"intervals": data["Intervals"]
}
for category in categories:
	sourceData[category] = data["BarData"][category]

plot = figure(
	x_range=data["Intervals"],
	plot_width=1800,
	plot_height=500,
	extra_y_ranges={"latency": Range1d(start=-20, end=18, min_interval=1.0)}
)

plot.vbar_stack(
	categories,
	x="intervals",
	width=1.0,
	source=sourceData,
	legend_label=list(map(lambda c: c + "  ", categories)),
	color=["#6d8ef9", "#7460e6", "#cc397e", "#eb6c2c", "#f4b23f", "#58595b"]
)


def latencyMap(latency):
	return list(map(lambda l: (math.log(l, 2) if l > 0 else 0), latency))


plot.line(
	data["Intervals"],
	latencyMap(data["LatencyReal"]),
	line_width=2,
	color="red",
	y_range_name="latency"
)

plot.line(
	data["Intervals"],
	latencyMap(data["LatencyIdeal"]),
	line_width=2,
	color="green",
	y_range_name="latency"
)

plot.xaxis.visible = False

plot.add_layout(LinearAxis(y_range_name="latency"), 'right')

plot.yaxis[1].formatter = FuncTickFormatter(code='''
return tick < 0 ? "" : 2 + tick.toString(10).split('').map(function (d) { return d === '-' ? '⁻' : (d === '.' ? '\u22c5' : '⁰¹²³⁴⁵⁶⁷⁸⁹'[+d]); }).join('');
''')
plot.add_tools(WheelZoomTool(dimensions="width"))
plot.add_tools(HoverTool(tooltips="$name: @$name"))

plot.legend.orientation = "horizontal"
plot.legend.location = "top_left"

plot.xgrid.grid_line_color = None

plot.output_backend = "svg"
export_svgs(plot, filename="plot.svg")

show(plot)
