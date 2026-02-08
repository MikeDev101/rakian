package menu

import (
	"context"
	"image"
	"log"
	"sync"
	"time"

	"keypad"
	"phone"
	"sh1107"
	"timers"
	"tones"
)

type MenuInstance interface {
	Run()                     // Starts the menu.
	Pause()                   // Exits the menu while retaining state, and can be resumed with Run().
	Stop()                    // Exits the menu and destroys any existing state.
	Configure()               // Required to be called before using Run(). Otherwise, a panic will occur.
	ConfigureWithArgs(...any) // Can be called anytime to passthrough arguments.
}

type Menu struct {
	Stack         []MenuInstance
	Menus         map[string]MenuInstance
	CurrentMenu   MenuInstance
	GlobalContext context.Context
	GlobalCancel  context.CancelFunc

	Display       *sh1107.SH1107
	Sprites       map[string]image.Image
	Modem         *phone.Modem
	KeypadEvents  <-chan *keypad.KeypadEvent
	Timers        map[string]*timers.ResettableTimer
	Player        *tones.Tones
	GlobalStorage map[string]any

	GlobalQuit func(uint8)

	lock        sync.RWMutex
	storageLock sync.Mutex
	masked      bool
}

// Mask sets a flag that prevents any menus from being pushed or popped.
// This is useful when a menu wants to temporarily block all other menus from being accessed.
// Note that this does not prevent the current menu from being stopped, nor does it prevent the global quit function from being called.
// Mask is automatically unset when the stack is empty.
func (m *Menu) Mask() {
	m.masked = true
}

func (m *Menu) ontop(menu string) bool {
	if len(m.Stack) == 0 {
		return false
	}
	return m.Stack[len(m.Stack)-1] == m.Menus[menu]
}

func (m *Menu) instack(menu string) bool {
	for _, mi := range m.Stack {
		if mi == m.Menus[menu] {
			return true
		}
	}
	return false
}

// Run the menu at the given index in the stack.
// If the menu is already running, do nothing.
// If the menu is not running, set the current menu to the given menu and run it in a goroutine.
// If the menu crashes with a panic, stop the player and render an alert to the user before returning to the home screen.
func (m *Menu) run(index int) {
	if m.CurrentMenu == m.Stack[index] {
		return
	}
	m.CurrentMenu = m.Stack[index]

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.Player.Stop()
				log.Printf("💥 Recovering from panic crash in goroutine %v", r)
				m.RenderAlert("alert", []string{"Crashed!", "Returing to", "the home", "screen."})
				go m.PlayAlert()
				time.Sleep(3 * time.Second)
				go m.ToStart()
			}
		}()

		m.CurrentMenu.Run()
	}()
}

// ToStart navigates to the home screen and stops all menus above it.
// This function is thread-safe and can be called from any goroutine.
func (m *Menu) ToStart() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	m.Stack[0].Configure()
	for len(m.Stack) > 1 {
		m.Stack[len(m.Stack)-1].Stop()
		m.Stack = m.Stack[:len(m.Stack)-1]
	}
	m.run(0)
}

// ToMenu navigates to a menu with the given name.
// If the menu does not exist, it will panic.
// ToMenu will stop the current menu and configure the new one.
// If the current menu is not the top-most menu, ToMenu will stop all menus above it.
// The target menu will then be pushed onto the stack and run.
func (m *Menu) ToMenu(menu string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	target := m.Menus[menu]
	if target == nil {
		return
	}
	target.Configure()
	if len(m.Stack) > 0 {
		m.Stack[len(m.Stack)-1].Stop()
	}
	m.Stack = append(m.Stack, target)
	m.run(len(m.Stack) - 1)
}

// PopToMenu pops the current menu and navigates to the given menu.
// If the given menu does not exist, it will panic.
// If the current menu is not the top-most menu, PopToMenu will stop all menus above it.
// The target menu will then be pushed onto the stack and run.
// If the current menu is already running, do nothing.
// PopToMenu is thread-safe and can be called from any goroutine.
func (m *Menu) PopToMenu(menu string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	target := m.Menus[menu]
	if target == nil {
		return
	}
	target.Configure()
	if len(m.Stack) > 0 {
		m.Stack[len(m.Stack)-1].Stop()
	}
	m.Stack = m.Stack[:len(m.Stack)-1]
	m.Stack = append(m.Stack, target)
	m.run(len(m.Stack) - 1)
}

// ToMenuWithArgs is similar to ToMenu, but it allows passing arguments to the target menu.
// If the menu does not have a ConfigureWithArgs method, it will panic.
func (m *Menu) ToMenuWithArgs(menu string, args ...any) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	target := m.Menus[menu]
	if target == nil {
		return
	}
	target.ConfigureWithArgs(args...)
	if len(m.Stack) > 0 {
		m.Stack[len(m.Stack)-1].Stop()
	}
	m.Stack = append(m.Stack, target)
	m.run(len(m.Stack) - 1)
}

func (m *Menu) PopToMenuWithArgs(menu string, args ...any) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	target := m.Menus[menu]
	if target == nil {
		return
	}
	target.ConfigureWithArgs(args...)
	if len(m.Stack) > 0 {
		m.Stack[len(m.Stack)-1].Stop()
	}
	m.Stack = m.Stack[:len(m.Stack)-1]
	m.Stack = append(m.Stack, target)
	m.run(len(m.Stack) - 1)
}

// Pushes a menu onto the stack and runs it.
// If the stack is empty, raises GlobalQuit.
func (m *Menu) Push(menu string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	target := m.Menus[menu]
	if target == nil {
		return
	}
	target.Configure()
	if m.CurrentMenu != nil {
		m.CurrentMenu.Pause()
	}
	m.Stack = append(m.Stack, target)
	m.run(len(m.Stack) - 1)
}

// Pushes a menu onto the stack and runs it with the given arguments.
// If the stack is empty, raises GlobalQuit.
func (m *Menu) PushWithArgs(menu string, args ...any) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	target := m.Menus[menu]
	if target == nil {
		return
	}
	target.ConfigureWithArgs(args...)
	if m.CurrentMenu != nil {
		m.CurrentMenu.Pause()
	}
	m.Stack = append(m.Stack, target)
	m.run(len(m.Stack) - 1)
}

// Pops the current menu off the stack and runs the previous menu.
// If the stack is empty, raises GlobalQuit.
func (m *Menu) Pop() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	if len(m.Stack) == 0 {
		return
	}

	// Pre-configure the next menu if it exists
	if len(m.Stack) > 1 {
		m.Stack[len(m.Stack)-2].Configure()
	}

	// Stop the current menu
	m.Stack[len(m.Stack)-1].Stop()

	// Pop the current menu
	m.Stack = m.Stack[:len(m.Stack)-1]

	if len(m.Stack) > 0 {
		m.run(len(m.Stack) - 1)
	} else {
		log.Println("⁉️ Stack is empty, raising GlobalQuit")
		m.CurrentMenu = nil
		m.GlobalQuit(3) // Perform a soft reboot since this shouldn't happen
	}
}

// Pops the current menu off the stack and runs the previous menu
// with the given arguments.
// If the stack is empty, raises GlobalQuit.
func (m *Menu) PopWithArgs(args ...any) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.masked {
		return
	}

	if len(m.Stack) == 0 {
		return
	}

	// Pre-configure the next menu if it exists
	if len(m.Stack) > 1 {
		m.Stack[len(m.Stack)-2].ConfigureWithArgs(args...)
	}

	// Stop the current menu
	m.Stack[len(m.Stack)-1].Stop()

	// Pop the current menu
	m.Stack = m.Stack[:len(m.Stack)-1]

	if len(m.Stack) > 0 {
		m.run(len(m.Stack) - 1)
	} else {
		log.Println("⁉️ Stack is empty, raising GlobalQuit")
		m.CurrentMenu = nil
		m.GlobalQuit(3) // Perform a soft reboot since this shouldn't happen
	}
}

func (m *Menu) Get(key string) any {
	m.storageLock.Lock()
	defer m.storageLock.Unlock()
	if value, ok := m.GlobalStorage[key]; !ok {
		return nil
	} else {
		return value
	}
}

func (m *Menu) Set(key string, value any) {
	m.storageLock.Lock()
	defer m.storageLock.Unlock()
	m.GlobalStorage[key] = value
}

func (m *Menu) Register(name string, instance MenuInstance) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.Menus == nil {
		m.Menus = make(map[string]MenuInstance)
	}
	m.Menus[name] = instance
}

func (m *Menu) Shutdown() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, menu := range m.Stack {
		menu.Stop()
	}
	for _, timer := range m.Timers {
		timer.Stop()
	}
	m.GlobalCancel()
}

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true // Completed successfully
	case <-time.After(timeout):
		return false // Timed out
	}
}

func Init(
	ctx context.Context,
	display *sh1107.SH1107,
	sprites map[string]image.Image,
	modem *phone.Modem,
	player *tones.Tones,
	globalquit func(uint8),
	keypadevents <-chan *keypad.KeypadEvent,
) *Menu {

	menu_ctx, menu_cancel := context.WithCancel(ctx)

	m := &Menu{
		GlobalContext: menu_ctx,
		GlobalCancel:  menu_cancel,
		Display:       display,
		Sprites:       sprites,
		Modem:         modem,
		KeypadEvents:  keypadevents,
		Timers:        make(map[string]*timers.ResettableTimer),
		Player:        player,
		GlobalStorage: make(map[string]any),
		GlobalQuit:    globalquit,
		masked:        false,
	}

	return m
}
