#!/usr/bin/env python3

from bokeh.plotting import figure, output_file, show
from bokeh.io import export_svgs
from bokeh.models import DatetimeTickFormatter, WheelZoomTool, HoverTool
from bokeh.palettes import Spectral6

from datetime import datetime as dt
import json
import math

with open("../usage.json") as f:
	data = json.load(f)

# categories = data["BarCategories"]
categories = ["transaction", "transaction-proposal", "endorsement", "non-revocation-request", "non-revocation-handle", "credentials"]
sourceData = {
	"intervals": data["Intervals"]
}
for category in categories:
	sourceData[category] = data["BarData"][category]

plot = figure(
	x_range=data["Intervals"],
	plot_width=1800,
	plot_height=500
)

plot.vbar_stack(
	categories,
	x="intervals",
	width=1.0,
	source=sourceData,
	legend_label=categories,
	color=Spectral6
)

latencyMap = lambda latency: list(map(lambda l: 8 + (math.log(l, 10) if l > 0 else 0), latency))

plot.line(
	data["Intervals"],
	latencyMap(data["LatencyReal"]),
	line_width=2,
	color="red"
)

plot.line(
	data["Intervals"],
	latencyMap(data["LatencyIdeal"]),
	line_width=2,
	color="green"
)

plot.xaxis.visible = False

# plot.extra_y_ranges = {"latency": Range1d(start=0, end=100)}
# p.circle(x, y2, color="blue", y_range_name="foo")
# p.add_layout(LinearAxis(y_range_name="foo"), 'left')

plot.add_tools(WheelZoomTool(dimensions="width"))
plot.add_tools(HoverTool(tooltips="$name @$name"))

# plot.output_backend = "svg"
# export_svgs(plot, filename="plot.svg")

show(plot)
