# wadjet

Wadjet is an experiment in building Slack bots (and possibly others) in a manner
that resembles multi-command CLI apps, using the same tools that one would use
for those apps (i.e., the flag package).

To add a command to wadjet, you add a Command function to the Commands map. The
Command function must then bind flags to the [FlagSet][] it's given, parse the
arguments it's given, and either return a [message][] or write its output to the
FlagSet's [output][FlagSet.Output]. When implementing sub-commands, you could
use either the first argument to do a similar command dispatch, a small libary
like [google/subcommands][], or some other library that doesn't presuppose the
existence of a terminal.

[FlagSet.Output]: https://golang.org/pkg/flag/#FlagSet.Output
[FlagSet]: https://godoc.org/flag#FlagSet
[google/subcommands]: https://github.com/google/subcommands
[slack.Msg]: https://godoc.org/github.com/nlopes/slack#Msg


# License

Wadjet is licensed under a two-clause BSD license.
