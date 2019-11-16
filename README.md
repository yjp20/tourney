# Tourney

Tourney is a discord bot that makes it easy to draft players inside Discord and manage teams. There are a few assumptions made with the bot, and that is that people will act according to the rules (i.e. this bot does _not_ enforce rules). The controls are very basic, and consist of the following commands:

* !help
* !new *tournament_name*
* !join [*tournament_name*] - join the tournament and make yourself available to draft
* !leave [*tournament_name*] - leaves the tournament
* !captain [*tournament_name*] - makes you a captain of a team in the tournament (creates new team)
* !draft *idx* [*tournament_name*] - makes player by the index *idx* your team
* !exit [*tournament_name*] - exits your team
* !status [*tournament_name*] - shows everything

Add this to your server with: https://discordapp.com/oauth2/authorize?&client_id=645225499471118357&scope=bot&permissions=11264
