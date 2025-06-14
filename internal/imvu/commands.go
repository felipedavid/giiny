package imvu

import (
	"fmt"
	"strings"
)

type IMVUCommand string

const (
	CmdBoot                IMVUCommand = "boot"
	CmdImvuShowGift        IMVUCommand = "imvu:showGift"
	CmdImvuSetRoomState    IMVUCommand = "imvu:setRoomState"
	CmdImvuChangeRoom      IMVUCommand = "imvu:changeRoom"
	CmdImvuGoto            IMVUCommand = "imvu:goto"
	CmdImvuIsPureUser      IMVUCommand = "imvu:isPureUser"
	CmdImvuTrigger         IMVUCommand = "imvu:trigger"
	CmdImvuUntrigger       IMVUCommand = "imvu:untrigger"
	CmdImvuActivateMusic   IMVUCommand = "imvu:activateMusic"
	CmdImvuDeactivateMusic IMVUCommand = "imvu:deactivateMusic"
	CmdImvuTry             IMVUCommand = "imvu:try"
	CmdImvuTryForUndo      IMVUCommand = "imvu:tryForUndo"
	CmdImvuRecommend       IMVUCommand = "imvu:recommend"
	CmdImvuPurchase        IMVUCommand = "imvu:purchase"
	CmdImvuGift            IMVUCommand = "imvu:gift"
	CmdImvuFlashCommand    IMVUCommand = "imvu:flashCommand"
	CmdMsg                 IMVUCommand = "msg"
	CmdHiResSnap           IMVUCommand = "hiResSnap"
	CmdHiResSnapLower      IMVUCommand = "hiressnap"
	CmdHiResNoBg           IMVUCommand = "hiResNoBg"
	CmdHiResNoBgLower      IMVUCommand = "hiresnobg"
	CmdUse                 IMVUCommand = "use"
	CmdPutOn               IMVUCommand = "putOn"
	CmdPutOnOutfit         IMVUCommand = "putOnOutfit"
	CmdTakeOff             IMVUCommand = "takeOff"
	CmdRemove              IMVUCommand = "remove"
	CmdRemoveMood          IMVUCommand = "removeMood"
	CmdResume              IMVUCommand = "resume"
	CmdAccept              IMVUCommand = "accept"
	CmdUid                 IMVUCommand = "uid"
	CmdUploadSnap          IMVUCommand = "uploadSnap"
	CmdSaveOutfit          IMVUCommand = "saveOutfit"
	CmdSnap                IMVUCommand = "snap"
	CmdSeat                IMVUCommand = "seat"
)

func (i *IMVU) Exec(command IMVUCommand, args ...string) error {
	i.SendChatMessage(fmt.Sprintf("*%s %s", command, strings.Join(args, " ")))

	return nil
}
