# abspitch-nv

This navidrome plugin overrides the `getSimilarSongs`, `getSimilarSongs2` and
`getSimilarArtists` endpoints to use the Absolute Pitch backend. It also implements the
SonicSimilarity capability.

## Installation

Download the `abspitch-nv.ndp` artifact from releases and put that under your plugins
dir.

You must also set this plugin as a top-priority metadata agent, if you set that
through an env variable it would something like
`ND_AGENTS=abspitch-nv,lastfm,deezer`.
