#!/usr/bin/env python3

from bokeh.plotting import figure, output_file, show
from bokeh.io import export_svgs
from bokeh.models import DatetimeTickFormatter

import pandas as pd
from datetime import datetime as dt
import json

with open("../usage.json") as f:
	data = json.load(f)

x = list(map(lambda point: pd.to_datetime(point["X"]), data))
y = list(map(lambda point: point["Y"], data))

plot = figure(
	x_axis_type="datetime",
	plot_width=1500,
	plot_height=300
)

formatterArgs = {}
for property in ["months", "days", "hours", "hourmin", "minutes", "minsec", "seconds", "milliseconds"]:
	formatterArgs[property] = ["%H:%M:%S.%3Ns"]
plot.xaxis.formatter = DatetimeTickFormatter(**formatterArgs)

# add a line renderer
plot.line(
	x=x,
	y=y,
	line_width=2
)

# plot.output_backend = "svg"
# export_svgs(plot, filename="plot.svg")

show(plot)
