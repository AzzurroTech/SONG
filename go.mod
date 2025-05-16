module SONG

go 1.24.3

replace SONG/vidi => ./vidi

replace SONG/veni => ./veni

replace SONG/connect_api => ./connect_api

replace SONG/delete_api => ./delete_api

replace SONG/get_api => ./get_api

replace SONG/head_api => ./head_api

replace SONG/options_api => ./options_api

replace SONG/patch_api => ./patch_api

replace SONG/put_api => ./put_api

replace SONG/trace_api => ./trace_api

require (
	SONG/connect_api v0.0.0-00010101000000-000000000000
	SONG/delete_api v0.0.0-00010101000000-000000000000
	SONG/get_api v0.0.0-00010101000000-000000000000
	SONG/head_api v0.0.0-00010101000000-000000000000
	SONG/options_api v0.0.0-00010101000000-000000000000
	SONG/patch_api v0.0.0-00010101000000-000000000000
	SONG/put_api v0.0.0-00010101000000-000000000000
	SONG/trace_api v0.0.0-00010101000000-000000000000
	SONG/veni v0.0.0-00010101000000-000000000000
	SONG/vidi v0.0.0-00010101000000-000000000000
)
