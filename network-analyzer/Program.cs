using System;
using System.Collections.Generic;
using System.ComponentModel.DataAnnotations;
using System.IO;
using System.Linq;
using System.Threading.Tasks;
using McMaster.Extensions.CommandLineUtils;
using Newtonsoft.Json;

namespace NetworkAnalyzer
{
	class Program
	{
		public static async Task<int> Main(string[] args) => await CommandLineApplication.ExecuteAsync<EntryPoint>(args);
	}

	[Command(Name = "network-analyzer", Description = "Utility to analyze network traffic after Fabric simulator", ThrowOnUnexpectedArgument = true)]
	class EntryPoint
	{
		[FileExists]
		[Required]
		[Option("--input <string>", Description = "JSON file with network log.")]
		public string InputFile { get; set; } = null;

		[Required]
		[DirectoryExists]
		[Option("--output <string>", Description = "Directory to write output files to.")]
		public string OutputDirectory { get; set; } = null;

		private async Task<int> OnExecute(CommandLineApplication app)
		{
			await Analyzer.AnalyzeAsync(InputFile, OutputDirectory);

			return 0;
		}
	}

	static class Analyzer
	{
		public class NetworkEvent
		{
			public string From { get; set; }
			public string To { get; set; }
			public string Object { get; set; }
			public int Size { get; set; }
			public DateTime Start { get; set; }
			public DateTime End { get; set; }
			public int LocalBandwidth { get; set; }
			public int GlobalBandwidth { get; set; }
			public long ID { get; set; }
		}

		class IntervalEndpoint
		{
			public string Object { get; set; }
			public DateTime When { get; set; }
			public bool Starts { get; set; }
			public TimeSpan Elapsed { get; set; }
			public long Id { get; set; }
			public int Size { get; set; }
		}

		class ChartData
		{
			public List<DateTime> Intervals { get; set; }

			public IEnumerable<string> BarCategories { get; set; }
			public Dictionary<string, List<int>> BarData { get; set; }

			public List<double> LatencyIdeal { get; set; }
			public List<double> LatencyReal { get; set; }
		}

		public static async Task AnalyzeAsync(string filePath, string dirPath)
		{
			var log = JsonConvert.DeserializeObject<IEnumerable<NetworkEvent>>(
				await File.ReadAllTextAsync(filePath)
			);

			Console.WriteLine($"Log size: {log.Count()}");

			await File.WriteAllTextAsync(Path.Combine(dirPath, "usage.json"), JsonConvert.SerializeObject(NetworkUsageStackedBarChart(log)));
		}

		private static ChartData NetworkUsageStackedBarChart(IEnumerable<NetworkEvent> log)
		{
			var result = new ChartData()
			{
				Intervals = new List<DateTime>(),
				LatencyReal = new List<double>(),
				LatencyIdeal = new List<double>()
			};

			var localBandwidth = log.First().LocalBandwidth;

			result.BarCategories = log.Select(e => e.Object).ToHashSet();
			result.BarData = result.BarCategories.ToDictionary(c => c, _ => new List<int>());

			Func<NetworkEvent, bool, IntervalEndpoint> toInterval =
				(e, starts) => new IntervalEndpoint
				{
					Object = e.Object,
					When = starts ? e.Start : e.End,
					Starts = starts,
					Elapsed = e.End - e.Start,
					Id = e.ID,
					Size = e.Size
				};

			var intervals = log
				.Select(e => new List<IntervalEndpoint> {
					toInterval(e, true),
					toInterval(e, false)
				})
				.SelectMany(i => i)
				.OrderBy(i => i.When);

			var timestamps = intervals.Select(i => i.When);

			var intervalSize = TimeSpan.FromMilliseconds(50);
			// (timestamps.Max() - timestamps.Min()) / 1000;

			Console.WriteLine($"Intervals number: {(timestamps.Max() - timestamps.Min()) / intervalSize}");

			var current = result.BarCategories.ToDictionary(c => c, c => 0);

			for (var cursor = timestamps.Min(); cursor < timestamps.Max(); cursor += intervalSize)
			{
				var inInterval = intervals.Where(i => i.When >= cursor && i.When <= cursor + intervalSize);

				foreach (var category in result.BarCategories)
				{
					current[category] += inInterval.Where(i => i.Object == category).Select(i => i.Starts ? +1 : -1).Sum();
					result.BarData[category].Add(current[category]);
				}

				// TODO include those started before and ending after current interval
				var latencies = inInterval.Select(i =>
				{
					var ideal = TimeSpan.FromMilliseconds(1000 * (double)i.Size / (double)localBandwidth);
					return (ideal: ideal, real: i.Elapsed);
				});

				Func<Func<(TimeSpan ideal, TimeSpan real), TimeSpan>, TimeSpan> avg =
					selector => TimeSpan.FromTicks(latencies.Count() > 0 ? Convert.ToInt64(latencies.Select(l => selector(l).Ticks).Average()) : 0);

				var ideal = avg(l => l.ideal);
				var real = avg(l => l.real);

				result.LatencyIdeal.Add(ideal.TotalMilliseconds);
				result.LatencyReal.Add(real.TotalMilliseconds);

				result.Intervals.Add(cursor);
			}

			return result;
		}
	}
}
