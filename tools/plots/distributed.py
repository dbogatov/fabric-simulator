#!/usr/bin/env python3

from bokeh.io import output_file, show, export_svgs
from bokeh.models import FactorRange, FuncTickFormatter
from bokeh.plotting import figure
from bokeh.transform import factor_cmap

categories = ["Extensions", "Endorsements", "Users", "Peers", "Minimal"]
factors = [
	("Extensions", "None"),
	("Extensions", "Auditing"),
	("Extensions", "Revocation"),
	("Extensions", "Both"),
	#
	("Endorsements", "1"),
	("Endorsements", "2"),
	("Endorsements", "3"),
	#
	("Users", "1"),
	("Users", "2"),
	("Users", "3"),
	("Users", "4"),
	("Users", "5"),
	#
	("Peers", "1"),
	("Peers", "2"),
	("Peers", "3"),
	#
	("Minimal", "None"),
	("Minimal", "Both"),
]

plot = figure(x_range=FactorRange(*factors), plot_height=250, toolbar_location=None, tools="")

x = [
	1161,
	1328,
	1425,
	1555,
	#
	1493,
	1555,
	1599,
	#
	1095,
	1213,
	1312,
	1496,
	1555,
	#
	1230,
	1339,
	1555,
	#
	599,
	855
]

# put your own colors in RGB hex format
colors = ["#6d8ef9", "#7460e6", "#cc397e", "#eb6c2c", "#f4b23f", "#58595b"]

plot.vbar(x=factors, fill_color=factor_cmap('x', palette=colors, factors=categories, end=1), top=x, width=0.9, alpha=0.5)

plot.y_range.start = 0
plot.x_range.range_padding = 0.1
plot.xaxis.major_label_orientation = 1
plot.xgrid.grid_line_color = None
plot.yaxis.formatter = FuncTickFormatter(code="""
	parts = tick.toString().split(".");
	parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, " ");
	return parts.join(".");
""")

plot.output_backend = "svg"
export_svgs(plot, filename="plot.svg")

show(plot)
