#!/usr/bin/env python3

from bokeh.plotting import figure, output_file, show
from bokeh.io import export_svgs
from bokeh.models import DatetimeTickFormatter, WheelZoomTool, HoverTool
from bokeh.palettes import Spectral8

import pandas as pd
from datetime import datetime as dt
import json

with open("../usage.json") as f:
	data = json.load(f)

# intervals = list(map(lambda point: pd.to_datetime(point), data["Intervals"]))
categories = list(data["Categories"])
sourceData = {
	"intervals": data["Intervals"]
}
for category in categories:
	sourceData[category] = data["Data"][category]

plot = figure(
	x_range=data["Intervals"],
	plot_width=1800,
	plot_height=500
)

formatterArgs = {}
for property in ["months", "days", "hours", "hourmin", "minutes", "minsec", "seconds", "milliseconds"]:
	formatterArgs[property] = ["%H:%M:%S.%3Ns"]
plot.xaxis.formatter = DatetimeTickFormatter(**formatterArgs)

plot.vbar_stack(
	categories,
	x="intervals",
	width=1.0,
	source=sourceData,
	legend_label=categories,
	color=Spectral8
)

plot.add_tools(WheelZoomTool(dimensions="width"))
plot.add_tools(HoverTool(tooltips="$name @$name"))

# plot.output_backend = "svg"
# export_svgs(plot, filename="plot.svg")

show(plot)
