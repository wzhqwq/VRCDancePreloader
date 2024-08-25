package gui

import (
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
				p.items[r.ID] = NewPlayItemGui(r)
				p.content.Add(p.items[r.ID].Card)
			}
		case item := <-updatePlayItemCh:
			if g, ok := p.items[item.GetInfo().ID]; ok {
				g.Update(item.Render())
			} else {
				updatePlayItemCh <- item
			}
		case items := <-removePlayItemCh:
			for _, item := range items {
				i := item.GetInfo()
				p.content.Remove(p.items[i.ID].Card)
				delete(p.items, i.ID)
			}
		}
	}
}
