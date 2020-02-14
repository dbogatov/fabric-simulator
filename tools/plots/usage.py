#!/usr/bin/env python3

from bokeh.plotting import figure, output_file, show
from bokeh.io import export_svgs
from bokeh.models import WheelZoomTool, HoverTool, Range1d, LinearAxis, FuncTickFormatter, Legend, Label
from bokeh.palettes import Spectral6

from datetime import datetime as dt
import json
import math

fontSizeTicks = "14pt"
fontSizeLabels = "12pt"
width=1800
height=500

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
	plot_width=width,
	plot_height=height,
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


real = plot.line(
	data["Intervals"],
	latencyMap(data["LatencyReal"]),
	line_width=2,
	color="red",
	y_range_name="latency"
)

ideal = plot.line(
	data["Intervals"],
	latencyMap(data["LatencyIdeal"]),
	line_width=2,
	color="green",
	y_range_name="latency"
)

plot.add_layout(LinearAxis(y_range_name="latency"), 'right')
plot.add_layout(Legend(items=[
	("Real latency (ms) ", [real]),
	("Ideal latency (ms) ", [ideal]),
]))

plot.yaxis[1].formatter = FuncTickFormatter(code='''
return tick < 0 ? "" : 2 + tick.toString(10).split('').map(function (d) { return d === '-' ? '⁻' : (d === '.' ? '\u22c5' : '⁰¹²³⁴⁵⁶⁷⁸⁹'[+d]); }).join('');
''')
plot.add_tools(WheelZoomTool(dimensions="width"))
plot.add_tools(HoverTool(tooltips="$name: @$name"))

plot.legend.orientation = "horizontal"

plot.legend[0].location = "top_left"
plot.legend[1].location = "top_right"
plot.legend.label_text_font_size = fontSizeLabels
plot.xaxis.axis_label_text_font_size = fontSizeLabels

plot.xaxis.visible = False
plot.add_layout(
	Label(
		x=(width / 2 - 180),
		y=0,
		x_units='screen',
		y_units='screen',
		text_font_style="italic",
		text='Intervals (20 milliseconds each)'
	)
)

plot.yaxis.major_label_text_font_size = fontSizeTicks
plot.yaxis.axis_label_text_font_size = fontSizeLabels
plot.yaxis[0].axis_label = "Number of objects in the network per interval"
plot.yaxis[1].axis_label = "Latency of the slowest object in milliseconds per interval"

plot.xgrid.grid_line_color = None

plot.output_backend = "svg"
export_svgs(plot, filename="plot.svg")

show(plot)
