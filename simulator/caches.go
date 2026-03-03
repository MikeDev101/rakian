package simulator

import (
	"gfx"
	"image"
)

// Caches
func (v *Simulator) GetAnimationCache(n string) *gfx.Animation      { return v.animationCache[n] }
func (v *Simulator) SetAnimationCache(n string, a *gfx.Animation)   { v.animationCache[n] = a }
func (v *Simulator) GetSpriteCache(n string) *image.Image           { return v.spriteCache[n] }
func (v *Simulator) SetSpriteCache(n string, i *image.Image)        { v.spriteCache[n] = i }
func (v *Simulator) GetSpriteCacheAll() map[string]*image.Image     { return v.spriteCache }
func (v *Simulator) GetFontCache(n string) map[rune]*image.Image    { return v.fontCache[n] }
func (v *Simulator) SetFontCache(n string, f map[rune]*image.Image) { v.fontCache[n] = f }
func (v *Simulator) Use_Font_Small_Bold() map[rune]*image.Image     { return gfx.Use_Font_Small_Bold(v) }
func (v *Simulator) Use_Font_Small_Plain() map[rune]*image.Image    { return gfx.Use_Font_Small_Plain(v) }
func (v *Simulator) Use_Font_Large_Bold() map[rune]*image.Image     { return gfx.Use_Font_Large_Bold(v) }
func (v *Simulator) Use_Font_Tiny() map[rune]*image.Image           { return gfx.Use_Font_Tiny(v) }
func (v *Simulator) Use_Sprites() map[string]*image.Image           { return gfx.Use_Sprites(v) }
