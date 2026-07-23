package downloader

//var managers = make(map[string]*downloadManager)
//
//// For future use
//
//func ListManagers() []string {
//	return []string{
//		"pypy",
//		"default",
//	}
//}
//
//func SubscribeManager(name string) *utils.EventSubscriber[ManagerChangeType] {
//	dm, ok := managers[name]
//	if !ok {
//		return nil
//	}
//	return dm.em.SubscribeEvent()
//}
//
//func GetQueue(name string) []*ManagedTask {
//	dm, ok := managers[name]
//	if !ok {
//		return nil
//	}
//	return dm.GetQueueSnapshot()
//}
