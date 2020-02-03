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
		}

		class IntervalEndpoint
		{
			public string Object { get; set; }
			public DateTime When { get; set; }
			public bool Starts { get; set; }
		}

		class StackedBarChartData
		{
			public IEnumerable<string> Categories { get; set; }
			public List<DateTime> Intervals { get; set; }
			public Dictionary<string, List<int>> Data { get; set; }
		}

		public static async Task AnalyzeAsync(string filePath, string dirPath)
		{
			var log = JsonConvert.DeserializeObject<IEnumerable<NetworkEvent>>(
				await File.ReadAllTextAsync(filePath)
			);

			Console.WriteLine($"Log size: {log.Count()}");

			await File.WriteAllTextAsync(Path.Combine(dirPath, "usage.json"), JsonConvert.SerializeObject(NetworkUsageStackedBarChart(log)));
		}

		private static StackedBarChartData NetworkUsageStackedBarChart(IEnumerable<NetworkEvent> log)
		{
			var result = new StackedBarChartData();

			result.Categories = log.Select(e => e.Object).ToHashSet();
			result.Data = result.Categories.ToDictionary(c => c, _ => new List<int>());
			result.Intervals = new List<DateTime>();

			var intervals = log
				.Select(e => new List<IntervalEndpoint> {
					new IntervalEndpoint {
						Object = e.Object,
						When = e.Start,
						Starts = true
					},
					new IntervalEndpoint {
						Object = e.Object,
						When = e.End,
						Starts = false
					}
				})
				.SelectMany(i => i)
				.OrderBy(i => i.When);

			var timestamps = intervals.Select(i => i.When);

			var intervalSize = (timestamps.Max() - timestamps.Min()) / 1000;

			Console.WriteLine($"Intervals number: {(timestamps.Max() - timestamps.Min()) / intervalSize}");

			var current = result.Categories.ToDictionary(c => c, c => 0);

			for (var cursor = timestamps.Min(); cursor < timestamps.Max(); cursor += intervalSize)
			{
				var inInterval = intervals.Where(i => i.When >= cursor && i.When <= cursor + intervalSize);

				foreach (var category in result.Categories)
				{
					current[category] += inInterval.Where(i => i.Object == category).Select(i => i.Starts ? +1 : -1).Sum();
					result.Data[category].Add(current[category]);
				}
				result.Intervals.Add(cursor);
			}

			return result;
		}
	}
}
