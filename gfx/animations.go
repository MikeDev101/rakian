package gfx

import "time"

// LoadAnimations loads animation sequences for display.
// Animations are stored in the driver's AnimationCache map for later retrieval.
func LoadAnimations(d Driver) {
	d.SetAnimationCache("boot", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "boot_0"},
			{Type: PLAY_FRAME, Image: "boot_1"},
			{Type: PLAY_FRAME, Image: "boot_2"},
			{Type: PLAY_FRAME, Image: "boot_3"},
			{Type: PLAY_FRAME, Image: "boot_4"},
			{Type: PLAY_FRAME, Image: "boot_5", DelayOverride: 500 * time.Millisecond},
			{Type: STOP_FRAME, Image: "boot_6"},
		},
		Loop:       false,
		FrameDelay: 250 * time.Millisecond,
	})

	d.SetAnimationCache("ok", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "ok_0"},
			{Type: PLAY_FRAME, Image: "ok_1"},
			{Type: PLAY_FRAME, Image: "ok_2"},
			{Type: STOP_FRAME, Image: "ok_3"},
		},
		Loop:       false,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("keypad_locked", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "keypad_locked_0"},
			{Type: PLAY_FRAME, Image: "keypad_locked_1"},
			{Type: PLAY_FRAME, Image: "keypad_locked_2"},
			{Type: STOP_FRAME, Image: "keypad_locked_3"},
		},
		Loop:       false,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("keypad_unlocked", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "keypad_locked_3"},
			{Type: PLAY_FRAME, Image: "keypad_locked_2"},
			{Type: PLAY_FRAME, Image: "keypad_locked_1"},
			{Type: STOP_FRAME, Image: "keypad_locked_0"},
		},
		Loop:       false,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("keypad_unlock_info", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "keypad_unlock_0"},
			{Type: PLAY_FRAME, Image: "keypad_unlock_1"},
			{Type: PLAY_FRAME, Image: "keypad_unlock_2"},
			{Type: PLAY_FRAME, Image: "keypad_unlock_3"},
			{Type: PLAY_FRAME, Image: "keypad_unlock_4"},
			{Type: PLAY_FRAME, Image: "keypad_unlock_5"},
			{Type: PLAY_FRAME, Image: "keypad_unlock_6", DelayOverride: 250 * time.Millisecond},
			{Type: PLAY_FRAME, Image: "keypad_unlock_0", DelayOverride: 500 * time.Millisecond},
			{Type: STOP_FRAME, Image: "keypad_unlock_6", DelayOverride: 500 * time.Millisecond},
		},
		Loop:       true,
		FrameDelay: 100 * time.Millisecond,
		LoopDelay:  1 * time.Second,
	})

	d.SetAnimationCache("low_battery", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "low_battery_0"},
			{Type: PLAY_FRAME, Image: "low_battery_1"},
			{Type: STOP_FRAME, Image: "low_battery_2"},
		},
		Loop:       true,
		FrameDelay: 750 * time.Millisecond,
	})

	d.SetAnimationCache("very_low_battery", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "very_low_battery_0"},
			{Type: PLAY_FRAME, Image: "very_low_battery_1"},
			{Type: PLAY_FRAME, Image: "very_low_battery_2"},
			{Type: PLAY_FRAME, Image: "very_low_battery_3", DelayOverride: 250 * time.Millisecond},
			{Type: PLAY_FRAME, Image: "very_low_battery_2"},
			{Type: PLAY_FRAME, Image: "very_low_battery_1"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
		LoopDelay:  250 * time.Millisecond,
	})

	d.SetAnimationCache("charging", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "charging_0"},
			{Type: PLAY_FRAME, Image: "charging_1"},
			{Type: PLAY_FRAME, Image: "charging_2"},
			{Type: STOP_FRAME, Image: "charging_3"},
		},
		Loop:       true,
		FrameDelay: 200 * time.Millisecond,
	})

	d.SetAnimationCache("vmail", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "vmail_0"},
			{Type: PLAY_FRAME, Image: "vmail_1"},
			{Type: PLAY_FRAME, Image: "vmail_2"},
			{Type: STOP_FRAME, Image: "vmail_3"},
		},
		Loop:       true,
		LoopDelay:  0 * time.Second,
		FrameDelay: 330 * time.Millisecond,
	})

	d.SetAnimationCache("sending", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "sending_0"},
			{Type: PLAY_FRAME, Image: "sending_1"},
			{Type: PLAY_FRAME, Image: "sending_2"},
			{Type: PLAY_FRAME, Image: "sending_3"},
			{Type: PLAY_FRAME, Image: "sending_4"},
			{Type: PLAY_FRAME, Image: "sending_5"},
			{Type: PLAY_FRAME, Image: "sending_6"},
			{Type: PLAY_FRAME, Image: "sending_7"},
			{Type: PLAY_FRAME, Image: "sending_8"},
			{Type: PLAY_FRAME, Image: "sending_9"},
			{Type: PLAY_FRAME, Image: "sending_10"},
			{Type: PLAY_FRAME, Image: "sending_11"},
			{Type: PLAY_FRAME, Image: "sending_12"},
			{Type: STOP_FRAME, Image: "sending_13"},
		},
		Loop:       false,
		FrameDelay: 50 * time.Millisecond,
	})

	d.SetAnimationCache("phonebook", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "phonebook_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "phonebook_1"},
			{Type: PLAY_FRAME, Image: "phonebook_2"},
			{Type: PLAY_FRAME, Image: "phonebook_3"},
			{Type: PLAY_FRAME, Image: "phonebook_0"},
			{Type: STOP_FRAME, Image: "phonebook_1"},
		},
		Loop:       true,
		FrameDelay: 250 * time.Millisecond,
	})

	d.SetAnimationCache("messages", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "messages_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "messages_0"},
			{Type: PLAY_FRAME, Image: "messages_1"},
			{Type: PLAY_FRAME, Image: "messages_2"},
			{Type: PLAY_FRAME, Image: "messages_3"},
			{Type: PLAY_FRAME, Image: "messages_4"},
			{Type: PLAY_FRAME, Image: "messages_5"},
			{Type: PLAY_FRAME, Image: "messages_6"},
			{Type: PLAY_FRAME, Image: "messages_7"},
			{Type: PLAY_FRAME, Image: "messages_8"},
			{Type: PLAY_FRAME, Image: "messages_9"},
			{Type: PLAY_FRAME, Image: "messages_10"},
			{Type: PLAY_FRAME, Image: "messages_11"},
			{Type: PLAY_FRAME, Image: "messages_12"},
			{Type: PLAY_FRAME, Image: "messages_13", DelayOverride: 1 * time.Second},
			{Type: PLAY_FRAME, Image: "messages_12"},
			{Type: PLAY_FRAME, Image: "messages_11"},
			{Type: STOP_FRAME, Image: "messages_0"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
		LoopDelay:  1 * time.Second,
	})

	d.SetAnimationCache("chats", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "chats_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "chats_0"},
			{Type: PLAY_FRAME, Image: "chats_1"},
			{Type: PLAY_FRAME, Image: "chats_2"},
			{Type: PLAY_FRAME, Image: "chats_3"},
			{Type: PLAY_FRAME, Image: "chats_4"},
			{Type: PLAY_FRAME, Image: "chats_5"},
			{Type: PLAY_FRAME, Image: "chats_6"},
			{Type: PLAY_FRAME, Image: "chats_7"},
			{Type: PLAY_FRAME, Image: "chats_8"},
			{Type: PLAY_FRAME, Image: "chats_9"},
			{Type: PLAY_FRAME, Image: "chats_10"},
			{Type: STOP_FRAME, Image: "chats_11"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("call_register", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "call_register_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "call_register_0"},
			{Type: PLAY_FRAME, Image: "call_register_1"},
			{Type: PLAY_FRAME, Image: "call_register_2"},
			{Type: PLAY_FRAME, Image: "call_register_3"},
			{Type: PLAY_FRAME, Image: "call_register_4"},
			{Type: PLAY_FRAME, Image: "call_register_5"},
			{Type: PLAY_FRAME, Image: "call_register_6"},
			{Type: PLAY_FRAME, Image: "call_register_7"},
			{Type: PLAY_FRAME, Image: "call_register_8"},
			{Type: PLAY_FRAME, Image: "call_register_9"},
			{Type: STOP_FRAME, Image: "call_register_10"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("settings", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "settings_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "settings_0"},
			{Type: PLAY_FRAME, Image: "settings_1"},
			{Type: PLAY_FRAME, Image: "settings_2"},
			{Type: PLAY_FRAME, Image: "settings_3"},
			{Type: PLAY_FRAME, Image: "settings_4"},
			{Type: PLAY_FRAME, Image: "settings_5"},
			{Type: PLAY_FRAME, Image: "settings_6"},
			{Type: STOP_FRAME, Image: "settings_7"},
		},
		Loop:       false,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("call_divert", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "call_divert_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "call_divert_0"},
			{Type: PLAY_FRAME, Image: "call_divert_1"},
			{Type: PLAY_FRAME, Image: "call_divert_2"},
			{Type: PLAY_FRAME, Image: "call_divert_3"},
			{Type: PLAY_FRAME, Image: "call_divert_4"},
			{Type: PLAY_FRAME, Image: "call_divert_5"},
			{Type: PLAY_FRAME, Image: "call_divert_6"},
			{Type: PLAY_FRAME, Image: "call_divert_7"},
			{Type: PLAY_FRAME, Image: "call_divert_8"},
			{Type: PLAY_FRAME, Image: "call_divert_9"},
			{Type: STOP_FRAME, Image: "call_divert_10"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("apps", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "apps_0"},
			{Type: PLAY_FRAME, Image: "apps_1"},
			{Type: PLAY_FRAME, Image: "apps_2"},
			{Type: STOP_FRAME, Image: "apps_3"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("calculator", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "calculator_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "calculator_0"},
			{Type: PLAY_FRAME, Image: "calculator_1"},
			{Type: PLAY_FRAME, Image: "calculator_2"},
			{Type: PLAY_FRAME, Image: "calculator_3"},
			{Type: PLAY_FRAME, Image: "calculator_4"},
			{Type: PLAY_FRAME, Image: "calculator_5"},
			{Type: PLAY_FRAME, Image: "calculator_6"},
			{Type: PLAY_FRAME, Image: "calculator_7"},
			{Type: PLAY_FRAME, Image: "calculator_8"},
			{Type: PLAY_FRAME, Image: "calculator_9"},
			{Type: PLAY_FRAME, Image: "calculator_10"},
			{Type: PLAY_FRAME, Image: "calculator_11"},
			{Type: STOP_FRAME, Image: "calculator_12"},
		},
		Loop:       true,
		FrameDelay: 500 * time.Millisecond,
		LoopDelay:  1 * time.Second,
	})

	d.SetAnimationCache("clock", &Animation{
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "clock_0"},
			{Type: PLAY_FRAME, Image: "clock_1"},
			{Type: PLAY_FRAME, Image: "clock_2"},
			{Type: PLAY_FRAME, Image: "clock_3"},
			{Type: PLAY_FRAME, Image: "clock_4"},
			{Type: PLAY_FRAME, Image: "clock_5"},
			{Type: PLAY_FRAME, Image: "clock_6"},
			{Type: STOP_FRAME, Image: "clock_7"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
	})

	d.SetAnimationCache("notes", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "notes_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "notes_0"},
			{Type: PLAY_FRAME, Image: "notes_1"},
			{Type: PLAY_FRAME, Image: "notes_2"},
			{Type: PLAY_FRAME, Image: "notes_3"},
			{Type: PLAY_FRAME, Image: "notes_4"},
			{Type: PLAY_FRAME, Image: "notes_5"},
			{Type: PLAY_FRAME, Image: "notes_6"},
			{Type: PLAY_FRAME, Image: "notes_7"},
			{Type: PLAY_FRAME, Image: "notes_8"},
			{Type: PLAY_FRAME, Image: "notes_9"},
			{Type: PLAY_FRAME, Image: "notes_10"},
			{Type: PLAY_FRAME, Image: "notes_11"},
			{Type: PLAY_FRAME, Image: "notes_12"},
			{Type: STOP_FRAME, Image: "notes_13"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
		LoopDelay:  1 * time.Second,
	})

	d.SetAnimationCache("tones", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "tones_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "tones_0"},
			{Type: PLAY_FRAME, Image: "tones_1"},
			{Type: PLAY_FRAME, Image: "tones_2"},
			{Type: PLAY_FRAME, Image: "tones_3"},
			{Type: PLAY_FRAME, Image: "tones_4"},
			{Type: PLAY_FRAME, Image: "tones_5"},
			{Type: PLAY_FRAME, Image: "tones_6"},
			{Type: PLAY_FRAME, Image: "tones_7"},
			{Type: PLAY_FRAME, Image: "tones_8"},
			{Type: PLAY_FRAME, Image: "tones_9"},
			{Type: PLAY_FRAME, Image: "tones_10"},
			{Type: PLAY_FRAME, Image: "tones_11"},
			{Type: PLAY_FRAME, Image: "tones_12"},
			{Type: PLAY_FRAME, Image: "tones_13"},
			{Type: PLAY_FRAME, Image: "tones_14"},
			{Type: PLAY_FRAME, Image: "tones_15"},
			{Type: STOP_FRAME, Image: "tones_16"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
		LoopDelay:  1 * time.Second,
	})

	d.SetAnimationCache("sim_tools", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "sim_tools_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "sim_tools_0"},
			{Type: PLAY_FRAME, Image: "sim_tools_1"},
			{Type: PLAY_FRAME, Image: "sim_tools_2"},
			{Type: PLAY_FRAME, Image: "sim_tools_3"},
			{Type: PLAY_FRAME, Image: "sim_tools_4"},
			{Type: PLAY_FRAME, Image: "sim_tools_5", DelayOverride: 1 * time.Second},
			{Type: PLAY_FRAME, Image: "sim_tools_4"},
			{Type: PLAY_FRAME, Image: "sim_tools_3"},
			{Type: PLAY_FRAME, Image: "sim_tools_2"},
			{Type: PLAY_FRAME, Image: "sim_tools_1"},
			{Type: STOP_FRAME, Image: "sim_tools_0"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
		LoopDelay:  1 * time.Second,
	})

	d.SetAnimationCache("drawing", &Animation{
		InitialFrame: &Frame{
			Type:  PLAY_FRAME,
			Image: "drawing_0",
		},
		InitialFrameDelay: 1 * time.Second,
		Frames: []Frame{
			{Type: PLAY_FRAME, Image: "drawing_0"},
			{Type: PLAY_FRAME, Image: "drawing_1"},
			{Type: PLAY_FRAME, Image: "drawing_2"},
			{Type: PLAY_FRAME, Image: "drawing_3"},
			{Type: PLAY_FRAME, Image: "drawing_4"},
			{Type: PLAY_FRAME, Image: "drawing_5"},
			{Type: PLAY_FRAME, Image: "drawing_6"},
			{Type: PLAY_FRAME, Image: "drawing_7"},
			{Type: STOP_FRAME, Image: "drawing_8"},
		},
		Loop:       true,
		FrameDelay: 150 * time.Millisecond,
		LoopDelay:  1 * time.Second,
	})
}
