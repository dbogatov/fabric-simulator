#!/usr/bin/env python3

from bokeh.plotting import figure, output_file, show
import pandas as pd
from datetime import datetime as dt
import json

with open("../usage.json") as f:
	data = json.load(f)

x = list(map(lambda point: pd.to_datetime(point["X"]), data))
y = list(map(lambda point: point["Y"], data))

output_file("line.html")

p = figure(
	x_axis_type="datetime",
	plot_width=1500,
	plot_height=300
)

# add a line renderer
p.line(
	x=x,
	y=y,
	line_width=2
)

show(p)
