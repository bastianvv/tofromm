# Blueprint

## startup
  -> register device -> get device_id

## TUI
  -> user browses/searches ROMs by platform
  -> user selects ROM list
  -> user confirms

## For each selected ROM:
  -> GET /api/roms/{id}/content/{file_name}    download ROM zip
  -> extract zip -> place in {roms_dir}/{platform_slug}/

## Open sync session
  -> POST /api/sync/negotiate (empty saves list to discover what server has)

## For each download operation:
  -> GET /api/saves/summary?rom_id=X           verify save exists
  -> Verify which save file is most recent, if local is newer, skip download
  -> GET /api/saves/{id}/content               download save file
  -> rename file to match ROM filename
  -> place in {saves_dir}/
  -> POST /api/saves/{id}/downloaded           confirm to server

## Close sync session
  -> POST /api/sync/sessions/{id}/complete
