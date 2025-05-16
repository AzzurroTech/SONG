module SONG

go 1.24.3

replace vidi => ./vidi

replace veni => ./veni

replace connect_api => ./connect_api

replace delete_api => ./delete_api

replace get_api => ./get_api

replace head_api => ./head_api

replace options_api => ./options_api

replace patch_api => ./patch_api

replace put_api => ./put_api

replace trace_api => ./trace_api

require (
	connect_api v0.0.0-00010101000000-000000000000
	delete_api v0.0.0-00010101000000-000000000000
	get_api v0.0.0-00010101000000-000000000000
	head_api v0.0.0-00010101000000-000000000000
	options_api v0.0.0-00010101000000-000000000000
	patch_api v0.0.0-00010101000000-000000000000
	put_api v0.0.0-00010101000000-000000000000
	trace_api v0.0.0-00010101000000-000000000000
	veni v0.0.0-00010101000000-000000000000
	vidi v0.0.0-00010101000000-000000000000
)
