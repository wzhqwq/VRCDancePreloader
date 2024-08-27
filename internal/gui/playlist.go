package gui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

type PlayListGui struct {
	Container *container.Scroll

	items   map[int]*PlayItemGui
	content *fyne.Container
}

type dynamicList struct {
}

func (d *dynamicList) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(playItemMinWidth, float32(len(objects))*(playItemHeight+theme.Padding())-theme.Padding())
}
func (d *dynamicList) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(fyne.NewSize(size.Width, playItemHeight))
	}
}

func NewPlayListGui() *PlayListGui {
	content := container.New(&dynamicList{})
	scroll := container.NewVScroll(container.NewPadded(content))
	scroll.SetMinSize(fyne.NewSize(playItemMinWidth+theme.Padding(), 400))

	return &PlayListGui{
		Container: scroll,

		items:   make(map[int]*PlayItemGui),
		content: content,
	}
}

func (p *PlayListGui) drawFromChannels() {
	for {
		select {
		case items := <-addPlayItemCh:
			for _, item := range items {
				r := item.Render()
				g := NewPlayItemGui(r)
				p.items[r.ID] = g
				p.content.Add(g.Card)
				p.content.Refresh()
				g.SlideIn()
			}
		case items := <-removePlayItemCh:
			for _, item := range items {
				i := item.GetInfo()
				if g, ok := p.items[i.ID]; ok {
					delete(p.items, i.ID)
					go func() {
						g.SlideOut()
						<-time.After(300 * time.Millisecond)
						p.content.Remove(g.Card)
					}()
				}
			}
		case item := <-updatePlayItemCh:
			if g, ok := p.items[item.GetInfo().ID]; ok {
				g.Update(item.Render())
			} else {
				updatePlayItemCh <- item
			}
		}
	}
}
