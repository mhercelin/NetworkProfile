# Privacy

This program will not transfer any information to other networked systems
unless specifically requested by the user or the person installing or operating
it.

More precisely, NetworkProfile:

- stores its profiles in a single YAML file under `%AppData%\NetworkProfile`,
  and writes a local log beside it;
- reads the network adapters of the machine it runs on, and the wireless
  networks Windows already knows, in order to display them;
- never sends anything anywhere: there is no telemetry, no update check, no
  crash reporting and no network request of its own. The fonts it displays are
  embedded in the executable rather than fetched, precisely so that it works on
  a machine whose connection is broken;
- uses no third-party service.

The only network traffic the program causes is the reconfiguration of the
adapter you asked it to change.
