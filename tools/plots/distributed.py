#!/usr/bin/env python3

from bokeh.io import output_file, show, export_svgs
from bokeh.models import FactorRange
from bokeh.plotting import figure

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

p = figure(x_range=FactorRange(*factors), plot_height=250, toolbar_location=None, tools="")

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
p.vbar(x=factors, top=x, width=0.9, alpha=0.5)

p.y_range.start = 0
p.x_range.range_padding = 0.1
p.xaxis.major_label_orientation = 1
p.xgrid.grid_line_color = None

p.output_backend = "svg"
export_svgs(p, filename="plot.svg")

show(p)
