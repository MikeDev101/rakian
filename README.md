<img width="433" height="85" alt="preview" src="https://github.com/user-attachments/assets/f298ce39-8bf7-46a4-8dcc-483036c4a403" />


**Rakian** is a Nokia Series 20-inspired Operating System built using Go.

> [!WARNING]
> This project is in an early alpha stage and is NOT ready for use.

# What?
This is a project that aims to build an entirely custom mobile RTOS for embedded ARM platforms (such as the Raspberry Pi) using common off-the-shelf components, and also because I got bored and I think it's cool.

# Ok... what's the current hardware?
For my experiments, I'm using the following hardware to develop the project:

* Raspberry Pi Zero 2W
* WaveShare SIM7600G-H LTE Cat.4 Modem
* SH1107 120x120 OLED display
* The guts and keypad of a Nokia 5110
* MAX17043 battery gauge
* MAX98357a I2S sound card w/ amplifier
* SPH0645 I2S microphone

The rest is a bunch of glue logic and power components.

# What's working?
* Sprites and animation
* Fonts
* Basic call functionality - Dial, Answer, Key-in during calls, End/Decline
* Menu navigation

# Work-in-progress
* Settings
* Network management (WiFi, Cellular)
* Applications/Games
* Phonebook
* Call Registry
* Call Redirection
* Clock (i.e. Alarms, Timers)
* Tones (custom composer, ringtone selector)

# Planned
* Generic platform builder?
* Device tree-based configuration
* Minimal initramfs for early-on controls
* Barebones Linux distro toolkit as a replacement for bloated Raspbian
