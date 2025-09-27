package foodgroup

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mk6i/retro-aim-server/state"
	"github.com/mk6i/retro-aim-server/wire"
)

func TestBARTService_UpsertItem(t *testing.T) {
	cases := []struct {
		// name is the unit test name
		name string
		// userSession is the session of the user adding to feedbag
		userSession *state.Session
		// inputSNAC is the SNAC sent from the client to the server
		inputSNAC wire.SNACMessage
		// mockParams is the list of params sent to mocks that satisfy this
		// method's dependencies
		mockParams mockParams
		// expectOutput is the SNAC sent from the server to client
		expectOutput wire.SNACMessage
	}{
		{
			name:        "upsert item",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x02_BARTUploadQuery{
					Type: 1,
					Data: []byte{'i', 't', 'e', 'm', 'd', 'a', 't', 'a'},
				},
			},
			mockParams: mockParams{
				bartItemManagerParams: bartItemManagerParams{
					bartItemManagerUpsertParams: bartItemManagerUpsertParams{
						{
							itemHash: []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							payload:  []byte{'i', 't', 'e', 'm', 'd', 'a', 't', 'a'},
							bartType: 1,
						},
					},
				},
				buddyBroadcasterParams: buddyBroadcasterParams{
					broadcastBuddyArrivedParams: broadcastBuddyArrivedParams{
						{
							screenName: state.DisplayScreenName("user_screen_name"),
						},
					},
				},
			},
			expectOutput: wire.SNACMessage{
				Frame: wire.SNACFrame{
					FoodGroup: wire.BART,
					SubGroup:  wire.BARTUploadReply,
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x03_BARTUploadReply{
					Code: wire.BARTReplyCodesSuccess,
					ID: wire.BARTID{
						Type: wire.BARTTypesBuddyIcon,
						BARTInfo: wire.BARTInfo{
							Flags: wire.BARTFlagsKnown,
							Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bartItemManager := newMockBARTItemManager(t)
			for _, params := range tc.mockParams.bartItemManagerUpsertParams {
				bartItemManager.EXPECT().
					InsertBARTItem(matchContext(), params.itemHash, params.payload, params.bartType).
					Return(nil)
			}
			buddyUpdateBroadcaster := newMockbuddyBroadcaster(t)
			for _, params := range tc.mockParams.broadcastBuddyArrivedParams {
				buddyUpdateBroadcaster.EXPECT().
					BroadcastBuddyArrived(mock.Anything, state.NewIdentScreenName(params.screenName.String()), mock.MatchedBy(func(userInfo wire.TLVUserInfo) bool {
						return userInfo.ScreenName == params.screenName.String()
					})).
					Return(params.err)
			}
			svc := NewBARTService(slog.Default(), bartItemManager, nil, nil, nil)
			svc.buddyUpdateBroadcaster = buddyUpdateBroadcaster

			output, err := svc.UpsertItem(context.Background(), tc.userSession, tc.inputSNAC.Frame,
				tc.inputSNAC.Body.(wire.SNAC_0x10_0x02_BARTUploadQuery))

			assert.NoError(t, err)
			assert.Equal(t, output, tc.expectOutput)
		})
	}
}

func TestBARTService_RetrieveItem(t *testing.T) {
	cases := []struct {
		// name is the unit test name
		name string
		// userSession is the session of the user adding to feedbag
		userSession *state.Session
		// inputSNAC is the SNAC sent from the client to the server
		inputSNAC wire.SNACMessage
		// mockParams is the list of params sent to mocks that satisfy this
		// method's dependencies
		mockParams mockParams
		// expectOutput is the SNAC sent from the server to client
		expectOutput wire.SNACMessage
		// expectErr is the expected error
		expectErr error
	}{
		{
			name:        "retrieve buddy icon",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x04_BARTDownloadQuery{
					ScreenName: "user_screen_name",
					Command:    1,
					BARTID: wire.BARTID{
						Type: wire.BARTTypesBuddyIcon,
						BARTInfo: wire.BARTInfo{
							Flags: wire.BARTFlagsKnown,
							Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
						},
					},
				},
			},
			mockParams: mockParams{
				bartItemManagerParams: bartItemManagerParams{
					bartItemManagerRetrieveParams: bartItemManagerRetrieveParams{
						{
							itemHash: []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							result:   []byte{'i', 't', 'e', 'm', 'd', 'a', 't', 'a'},
						},
					},
				},
			},
			expectOutput: wire.SNACMessage{
				Frame: wire.SNACFrame{
					FoodGroup: wire.BART,
					SubGroup:  wire.BARTDownloadReply,
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x05_BARTDownloadReply{
					ScreenName: "user_screen_name",
					BARTID: wire.BARTID{
						Type: wire.BARTTypesBuddyIcon,
						BARTInfo: wire.BARTInfo{
							Flags: wire.BARTFlagsKnown,
							Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
						},
					},
					Data: []byte{'i', 't', 'e', 'm', 'd', 'a', 't', 'a'},
				},
			},
		},
		{
			name:        "retrieve blank icon used for clearing buddy icon",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x04_BARTDownloadQuery{
					ScreenName: "user_screen_name",
					Command:    1,
					BARTID: wire.BARTID{
						Type: wire.BARTTypesBuddyIcon,
						BARTInfo: wire.BARTInfo{
							Flags: wire.BARTFlagsKnown,
							Hash:  wire.GetClearIconHash(),
						},
					},
				},
			},
			expectOutput: wire.SNACMessage{
				Frame: wire.SNACFrame{
					FoodGroup: wire.BART,
					SubGroup:  wire.BARTDownloadReply,
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x05_BARTDownloadReply{
					ScreenName: "user_screen_name",
					BARTID: wire.BARTID{
						Type: wire.BARTTypesBuddyIcon,
						BARTInfo: wire.BARTInfo{
							Flags: wire.BARTFlagsKnown,
							Hash:  wire.GetClearIconHash(),
						},
					},
					Data: blankGIF,
				},
			},
		},
		{
			name:        "retrieve item with error",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x04_BARTDownloadQuery{
					ScreenName: "user_screen_name",
					Command:    1,
					BARTID: wire.BARTID{
						Type: wire.BARTTypesBuddyIcon,
						BARTInfo: wire.BARTInfo{
							Flags: wire.BARTFlagsKnown,
							Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
						},
					},
				},
			},
			mockParams: mockParams{
				bartItemManagerParams: bartItemManagerParams{
					bartItemManagerRetrieveParams: bartItemManagerRetrieveParams{
						{
							itemHash: []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							result:   nil,
							err:      assert.AnError,
						},
					},
				},
			},
			expectOutput: wire.SNACMessage{},
			expectErr:    assert.AnError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bartItemManager := newMockBARTItemManager(t)
			for _, params := range tc.mockParams.bartItemManagerParams.bartItemManagerRetrieveParams {
				bartItemManager.EXPECT().
					BARTItem(matchContext(), params.itemHash).
					Return(params.result, params.err)
			}

			svc := NewBARTService(slog.Default(), bartItemManager, nil, nil, nil)

			output, err := svc.RetrieveItem(context.Background(), tc.inputSNAC.Frame, tc.inputSNAC.Body.(wire.SNAC_0x10_0x04_BARTDownloadQuery))

			assert.ErrorIs(t, err, tc.expectErr)
			assert.Equal(t, output, tc.expectOutput)
		})
	}
}

func TestBARTService_RetrieveItemV2(t *testing.T) {
	cases := []struct {
		// name is the unit test name
		name string
		// userSession is the session of the user adding to feedbag
		userSession *state.Session
		// inputSNAC is the SNAC sent from the client to the server
		inputSNAC wire.SNACMessage
		// mockParams is the list of params sent to mocks that satisfy this
		// method's dependencies
		mockParams mockParams
		// expectOutput is the SNAC sent from the server to client
		expectOutput []wire.SNACMessage
		// expectErr is the expected error
		expectErr error
	}{
		{
			name:        "retrieve single buddy icon",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x06_BARTDownload2Query{
					ScreenName: "user_screen_name",
					IDs: []wire.BARTID{
						{
							Type: wire.BARTTypesBuddyIcon,
							BARTInfo: wire.BARTInfo{
								Flags: wire.BARTFlagsKnown,
								Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							},
						},
					},
				},
			},
			mockParams: mockParams{
				bartItemManagerParams: bartItemManagerParams{
					bartItemManagerRetrieveParams: bartItemManagerRetrieveParams{
						{
							itemHash: []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							result:   []byte{'i', 't', 'e', 'm', 'd', 'a', 't', 'a'},
						},
					},
				},
			},
			expectOutput: []wire.SNACMessage{
				{
					Frame: wire.SNACFrame{
						FoodGroup: wire.BART,
						SubGroup:  wire.BARTDownload2Reply,
						RequestID: 1234,
					},
					Body: wire.SNAC_0x10_0x07_BARTDownload2Reply{
						ScreenName: "user_screen_name",
						ReplyID: wire.BartQueryReplyID{
							QueryID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
								},
							},
							Code: 0x00, // found
							ReplyID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
								},
							},
						},
						Data: []byte{'i', 't', 'e', 'm', 'd', 'a', 't', 'a'},
					},
				},
			},
		},
		{
			name:        "retrieve multiple buddy icons",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x06_BARTDownload2Query{
					ScreenName: "user_screen_name",
					IDs: []wire.BARTID{
						{
							Type: wire.BARTTypesBuddyIcon,
							BARTInfo: wire.BARTInfo{
								Flags: wire.BARTFlagsKnown,
								Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							},
						},
						{
							Type: wire.BARTTypesBuddyIcon,
							BARTInfo: wire.BARTInfo{
								Flags: wire.BARTFlagsKnown,
								Hash:  []byte{0x5f, 0xea, 0xd2, 0xa7, 0x56, 0xec, 0x6b, 0xfd, 0xec, 0x06, 0xd8, 0xb3, 0x5f, 0x9f, 0xb1, 0xfe},
							},
						},
					},
				},
			},
			mockParams: mockParams{
				bartItemManagerParams: bartItemManagerParams{
					bartItemManagerRetrieveParams: bartItemManagerRetrieveParams{
						{
							itemHash: []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							result:   []byte{'f', 'i', 'r', 's', 't', 'i', 'c', 'o', 'n'},
						},
						{
							itemHash: []byte{0x5f, 0xea, 0xd2, 0xa7, 0x56, 0xec, 0x6b, 0xfd, 0xec, 0x06, 0xd8, 0xb3, 0x5f, 0x9f, 0xb1, 0xfe},
							result:   []byte{'s', 'e', 'c', 'o', 'n', 'd', 'i', 'c', 'o', 'n'},
						},
					},
				},
			},
			expectOutput: []wire.SNACMessage{
				{
					Frame: wire.SNACFrame{
						FoodGroup: wire.BART,
						SubGroup:  wire.BARTDownload2Reply,
						RequestID: 1234,
					},
					Body: wire.SNAC_0x10_0x07_BARTDownload2Reply{
						ScreenName: "user_screen_name",
						ReplyID: wire.BartQueryReplyID{
							QueryID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
								},
							},
							Code: 0x00, // found
							ReplyID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
								},
							},
						},
						Data: []byte{'f', 'i', 'r', 's', 't', 'i', 'c', 'o', 'n'},
					},
				},
				{
					Frame: wire.SNACFrame{
						FoodGroup: wire.BART,
						SubGroup:  wire.BARTDownload2Reply,
						RequestID: 1234,
					},
					Body: wire.SNAC_0x10_0x07_BARTDownload2Reply{
						ScreenName: "user_screen_name",
						ReplyID: wire.BartQueryReplyID{
							QueryID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  []byte{0x5f, 0xea, 0xd2, 0xa7, 0x56, 0xec, 0x6b, 0xfd, 0xec, 0x06, 0xd8, 0xb3, 0x5f, 0x9f, 0xb1, 0xfe},
								},
							},
							Code: 0x00, // found
							ReplyID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  []byte{0x5f, 0xea, 0xd2, 0xa7, 0x56, 0xec, 0x6b, 0xfd, 0xec, 0x06, 0xd8, 0xb3, 0x5f, 0x9f, 0xb1, 0xfe},
								},
							},
						},
						Data: []byte{'s', 'e', 'c', 'o', 'n', 'd', 'i', 'c', 'o', 'n'},
					},
				},
			},
		},
		{
			name:        "retrieve single item with error",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x06_BARTDownload2Query{
					ScreenName: "user_screen_name",
					IDs: []wire.BARTID{
						{
							Type: wire.BARTTypesBuddyIcon,
							BARTInfo: wire.BARTInfo{
								Flags: wire.BARTFlagsKnown,
								Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							},
						},
					},
				},
			},
			mockParams: mockParams{
				bartItemManagerParams: bartItemManagerParams{
					bartItemManagerRetrieveParams: bartItemManagerRetrieveParams{
						{
							itemHash: []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							result:   nil,
							err:      assert.AnError,
						},
					},
				},
			},
			expectOutput: nil,
			expectErr:    assert.AnError,
		},
		{
			name:        "retrieve multiple items with error on second item",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x06_BARTDownload2Query{
					ScreenName: "user_screen_name",
					IDs: []wire.BARTID{
						{
							Type: wire.BARTTypesBuddyIcon,
							BARTInfo: wire.BARTInfo{
								Flags: wire.BARTFlagsKnown,
								Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							},
						},
						{
							Type: wire.BARTTypesBuddyIcon,
							BARTInfo: wire.BARTInfo{
								Flags: wire.BARTFlagsKnown,
								Hash:  []byte{0x5f, 0xea, 0xd2, 0xa7, 0x56, 0xec, 0x6b, 0xfd, 0xec, 0x06, 0xd8, 0xb3, 0x5f, 0x9f, 0xb1, 0xfe},
							},
						},
					},
				},
			},
			mockParams: mockParams{
				bartItemManagerParams: bartItemManagerParams{
					bartItemManagerRetrieveParams: bartItemManagerRetrieveParams{
						{
							itemHash: []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							result:   []byte{'f', 'i', 'r', 's', 't', 'i', 'c', 'o', 'n'},
							err:      nil,
						},
						{
							itemHash: []byte{0x5f, 0xea, 0xd2, 0xa7, 0x56, 0xec, 0x6b, 0xfd, 0xec, 0x06, 0xd8, 0xb3, 0x5f, 0x9f, 0xb1, 0xfe},
							result:   nil,
							err:      assert.AnError,
						},
					},
				},
			},
			expectOutput: nil,
			expectErr:    assert.AnError,
		},
		{
			name:        "retrieve mixed items (regular icon and clear icon)",
			userSession: newTestSession("user_screen_name"),
			inputSNAC: wire.SNACMessage{
				Frame: wire.SNACFrame{
					RequestID: 1234,
				},
				Body: wire.SNAC_0x10_0x06_BARTDownload2Query{
					ScreenName: "user_screen_name",
					IDs: []wire.BARTID{
						{
							Type: wire.BARTTypesBuddyIcon,
							BARTInfo: wire.BARTInfo{
								Flags: wire.BARTFlagsKnown,
								Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							},
						},
						{
							Type: wire.BARTTypesBuddyIcon,
							BARTInfo: wire.BARTInfo{
								Flags: wire.BARTFlagsKnown,
								Hash:  wire.GetClearIconHash(),
							},
						},
					},
				},
			},
			mockParams: mockParams{
				bartItemManagerParams: bartItemManagerParams{
					bartItemManagerRetrieveParams: bartItemManagerRetrieveParams{
						{
							itemHash: []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
							result:   []byte{'r', 'e', 'g', 'u', 'l', 'a', 'r', 'i', 'c', 'o', 'n'},
						},
					},
				},
			},
			expectOutput: []wire.SNACMessage{
				{
					Frame: wire.SNACFrame{
						FoodGroup: wire.BART,
						SubGroup:  wire.BARTDownload2Reply,
						RequestID: 1234,
					},
					Body: wire.SNAC_0x10_0x07_BARTDownload2Reply{
						ScreenName: "user_screen_name",
						ReplyID: wire.BartQueryReplyID{
							QueryID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
								},
							},
							Code: 0x00, // found
							ReplyID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  []byte{0x4e, 0xd9, 0xc1, 0x96, 0x45, 0xdb, 0x5a, 0xec, 0xdb, 0xf5, 0xc7, 0xa2, 0x4e, 0x8e, 0xa0, 0xed},
								},
							},
						},
						Data: []byte{'r', 'e', 'g', 'u', 'l', 'a', 'r', 'i', 'c', 'o', 'n'},
					},
				},
				{
					Frame: wire.SNACFrame{
						FoodGroup: wire.BART,
						SubGroup:  wire.BARTDownload2Reply,
						RequestID: 1234,
					},
					Body: wire.SNAC_0x10_0x07_BARTDownload2Reply{
						ScreenName: "user_screen_name",
						ReplyID: wire.BartQueryReplyID{
							QueryID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  wire.GetClearIconHash(),
								},
							},
							Code: 0x00, // found
							ReplyID: wire.BARTID{
								Type: wire.BARTTypesBuddyIcon,
								BARTInfo: wire.BARTInfo{
									Flags: wire.BARTFlagsKnown,
									Hash:  wire.GetClearIconHash(),
								},
							},
						},
						Data: blankGIF,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bartItemManager := newMockBARTItemManager(t)
			for _, params := range tc.mockParams.bartItemManagerParams.bartItemManagerRetrieveParams {
				bartItemManager.EXPECT().
					BARTItem(matchContext(), params.itemHash).
					Return(params.result, params.err)
			}

			svc := NewBARTService(slog.Default(), bartItemManager, nil, nil, nil)

			output, err := svc.RetrieveItemV2(context.Background(), tc.inputSNAC.Frame, tc.inputSNAC.Body.(wire.SNAC_0x10_0x06_BARTDownload2Query))

			assert.ErrorIs(t, err, tc.expectErr)
			assert.Equal(t, output, tc.expectOutput)
		})
	}
}
