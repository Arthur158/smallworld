# In-game wiki

See [Controls](CONTROLS.md) for the complete mouse and keyboard guide.

Hold Ctrl (Cmd on macOS) and click a race or trait card, a piece stack, or a map region. Alternatively, enable **Inspect** above the board; Escape exits it. **Game wiki** opens the searchable library from the lobby or a game. Inspection never sends a game command. The board continues running while the wiki is open; region pages show the clicked snapshot.

The guide separates races, traits, powers (relics and Popular Places), pieces, and terrain. A tribe page shows its race and trait together with related pieces. Token pages resolve active and declined pairings from the current game. Published rules are paraphrased and linked; known differences in this implementation are explained.

## Maintaining content

- `src/data/wikiRaces.ts` and `wikiTraits.ts`: race and trait descriptions.
- `src/data/wikiCatalog.ts`: powers, special markers, terrain and related-piece mappings.
- `python3 scripts/generate-wiki-metadata.py`: refresh artwork names and built-in terrain metadata after changing assets or maps. Generated maps use the terrain supplied by the server.
- `CI=true npm test -- --watchAll=false --watchman=false --runInBand`: run frontend checks, including inspection isolation, article navigation, category coverage and map metadata.

The feature is entirely frontend-side and does not alter game commands, rules, snapshots, or AI input.
