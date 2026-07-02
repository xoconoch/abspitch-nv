# NMFT-NV

This navidrome plugin overrides the `getSimilarSongs`, `getSimilarSongs2` and
`getSimilarArtists` endpoints to use the NMFT backend. It also implements the
SonicSimilarity capability.

## Installation

Download the `nmft-nv.ndp` artifact from releases and put that under your plugins
dir.

You must also set this plugin as a top-priority metadata agent, if you set that
through an env variable it would something like
`ND_AGENTS=nmft-nv,lastfm,deezer`.
