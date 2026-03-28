package config

// DefaultConf is the default bujotui.conf content written by `bujotui config init`.
const DefaultConf = `[symbols]
task = .
done = x
migrated = >
scheduled = <
cancelled = X
event = o
note = -
idea = *
urgent = !
waiting = ~
health = +

[transitions]
task = done, migrated, scheduled, cancelled
event =
note =
idea = done, migrated, cancelled
urgent = done, migrated, cancelled
waiting = done, migrated, cancelled
health =
done =
migrated =
scheduled = done, cancelled
cancelled =

[projects]
inbox

[people]
self

[colors]
done = green
cancelled = red
migrated = blue
scheduled = cyan

[defaults]
project = inbox
person = self
symbol = task
`
