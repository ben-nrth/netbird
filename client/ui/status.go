package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	log "github.com/sirupsen/logrus"

	"github.com/netbirdio/netbird/client/internal"
	"github.com/netbirdio/netbird/client/proto"
)

func (s *serviceClient) showStatusWindowUI() {
	// add status window widgets
	s.wStatusWindow = s.app.NewWindow("NetBird Status")
	s.wStatusWindow.SetOnClosed(s.cancel)

	s.connectStatus = widget.NewLabel("")

	s.connectButton = widget.NewButton("Connect", func() {
		err := s.onConnectButtonClicked()
		if err != nil {
			log.Errorf("Status button click failed with error: %v", err)
		}
	})

	vbox := container.NewVBox()

	status := container.New(layout.NewGridLayout(2), widget.NewLabel("Netbird Status:"), s.connectStatus)

	vbox.Add(status)
	vbox.Add(s.connectButton)

	s.wStatusWindow.SetContent(vbox)
	s.wStatusWindow.Resize(fyne.NewSize(300, 80))
	s.wStatusWindow.SetFixedSize(true)
	s.wStatusWindow.Show()

	s.updateStatusWindow()
	s.startAutoRefreshStatus(10 * time.Second)
}

func (s *serviceClient) onConnectButtonClicked() error {
	if s.connectButton.Text == "Connect" {
		err := s.onConnect()
		if err != nil {
			return err
		}
	} else {
		err := s.onDisconnect()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *serviceClient) onConnect() error {
	s.connectButton.Disable()
	s.setStatusMessage("Connecting...")
	defer s.updateStatusWindow()

	conn, err := s.getSrvClient(defaultFailTimeout)
	if err != nil {
		log.Errorf("get client: %v", err)
		return err
	}

	err = s.login()
	if err != nil {
		log.Errorf("login failed with: %v", err)
		return err
	}

	status, err := conn.Status(s.ctx, &proto.StatusRequest{})
	if err != nil {
		log.Errorf("get service status: %v", err)
		return err
	}

	if status.Status == string(internal.StatusConnected) {
		log.Warnf("already connected")
		return nil
	}

	if _, err := s.conn.Up(s.ctx, &proto.UpRequest{}); err != nil {
		log.Errorf("up service: %v", err)
		return err
	}

	return nil
}

func (s *serviceClient) setStatusMessage(msg string) {
	s.connectStatus.Text = msg
	s.connectStatus.Refresh()
}

func (s *serviceClient) onDisconnect() error {
	s.connectButton.Disable()
	s.setStatusMessage("Disconnecting...")
	defer s.updateStatusWindow()

	conn, err := s.getSrvClient(defaultFailTimeout)
	if err != nil {
		s.setStatusMessage("Disconnect failed.")

		log.Errorf("get client: %v", err)
		return err
	}

	status, err := conn.Status(s.ctx, &proto.StatusRequest{})
	if err != nil {
		s.setStatusMessage("Disconnect failed.")

		log.Errorf("get service status: %v", err)
		return err
	}

	if status.Status != string(internal.StatusConnected) && status.Status != string(internal.StatusConnecting) {
		s.setStatusMessage("Already connected.")

		log.Warnf("already down")
		return nil
	}

	if _, err := s.conn.Down(s.ctx, &proto.DownRequest{}); err != nil {
		s.setStatusMessage("Disconnect failed.")

		log.Errorf("down service: %v", err)
		return err
	}

	return nil
}

func (s *serviceClient) startAutoRefreshStatus(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.updateStatusWindow()
		}
	}()

	s.wStatusWindow.SetOnClosed(func() {
		ticker.Stop()
		s.cancel()
	})
}

func (s *serviceClient) updateStatusWindow() error {
	defer s.wStatusWindow.Content().Refresh()

	conn, err := s.getSrvClient(defaultFailTimeout)
	if err != nil {
		log.Errorf("get service client: %v", err)
		return err
	}

	status, err := conn.Status(s.ctx, &proto.StatusRequest{})
	if err != nil {
		log.Errorf("get service status: %v", err)
		return err
	}

	switch status.Status {
	case string(internal.StatusConnected):
		s.connectStatus.Text = "Connected"
		s.connectButton.Text = "Disconnect"

		if s.connectButton.Disabled() {
			s.connectButton.Enable()
		}

	case string(internal.StatusConnecting):
		s.connectStatus.Text = "Connecting..."
		s.connectButton.Text = "Connecting..."

		if !s.connectButton.Disabled() {
			s.connectButton.Disable()
		}

	case string(internal.StatusIdle):
		s.connectStatus.Text = "Disconnected"
		s.connectButton.Text = "Connect"

		if s.connectButton.Disabled() {
			s.connectButton.Enable()
		}

	case string(internal.StatusLoginFailed):
		s.connectStatus.Text = "Login failed"
		s.connectButton.Text = "Connect"

		if s.connectButton.Disabled() {
			s.connectButton.Enable()
		}

	case string(internal.StatusNeedsLogin):
		s.connectStatus.Text = "Need to login"
		s.connectButton.Text = "Connect"

		if s.connectButton.Disabled() {
			s.connectButton.Enable()
		}
	}

	return nil
}
