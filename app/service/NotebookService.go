package service

import (
//	"fmt"
	"gopkg.in/mgo.v2/bson"
	"github.com/leanote/leanote/app/db"
	"github.com/leanote/leanote/app/info"
	. "github.com/leanote/leanote/app/lea"
	"sort"
	"time"
)

// 笔记本

type NotebookService struct {
}

// 排序
func sortSubNotebooks(eachNotebooks info.SubNotebooks) info.SubNotebooks {
	// 遍历子, 则子往上进行排序
	for _, eachNotebook := range eachNotebooks {
		if eachNotebook.Subs != nil && len(eachNotebook.Subs) > 0 {
			eachNotebook.Subs = sortSubNotebooks(eachNotebook.Subs)
		}
	}

	// 子排完了, 本层排
	sort.Sort(&eachNotebooks)
	return eachNotebooks
}

// 整理(成有关系)并排序
// GetNotebooks()调用
// ShareService调用
func ParseAndSortNotebooks(userNotebooks []info.Notebook, noParentDelete, needSort bool) info.SubNotebooks {
	// 整理成info.Notebooks
	// 第一遍, 建map
	// notebookId => info.Notebooks
	userNotebooksMap := make(map[bson.ObjectId]*info.Notebooks, len(userNotebooks))
	for _, each := range userNotebooks {
		newNotebooks := info.Notebooks{Subs: info.SubNotebooks{}}
		newNotebooks.NotebookId = each.NotebookId
		newNotebooks.Title = each.Title
		newNotebooks.Seq = each.Seq
		newNotebooks.UserId = each.UserId
		newNotebooks.ParentNotebookId = each.ParentNotebookId
		newNotebooks.NumberNotes = each.NumberNotes
		newNotebooks.IsTrash = each.IsTrash
		newNotebooks.IsBlog = each.IsBlog
		
		// 存地址
		userNotebooksMap[each.NotebookId] = &newNotebooks
	}

	// 第二遍, 追加到父下
	
	// 需要删除的id
	needDeleteNotebookId := map[bson.ObjectId]bool{}
	for id, each := range userNotebooksMap {
		// 如果有父, 那么追加到父下, 并剪掉当前, 那么最后就只有根的元素
		if each.ParentNotebookId.Hex() != "" {
			if userNotebooksMap[each.ParentNotebookId] != nil {
				userNotebooksMap[each.ParentNotebookId].Subs = append(userNotebooksMap[each.ParentNotebookId].Subs, each) // Subs是存地址
				// 并剪掉
				// bug
				needDeleteNotebookId[id] = true
				// delete(userNotebooksMap, id)
			} else if noParentDelete {
				// 没有父, 且设置了要删除	
				needDeleteNotebookId[id] = true
				// delete(userNotebooksMap, id)
			}
		}
	}

	// 第三遍, 得到所有根
	final := make(info.SubNotebooks, len(userNotebooksMap)-len(needDeleteNotebookId))
	i := 0
	for id, each := range userNotebooksMap {
		if !needDeleteNotebookId[id] {
			final[i] = each
			i++
		}
	}

	// 最后排序
	if needSort {
		return sortSubNotebooks(final)
	}
	return final
}

// 得到某notebook
func (this *NotebookService) GetNotebook(notebookId, userId string) info.Notebook {
	notebook := info.Notebook{}
	db.GetByIdAndUserId(db.Notebooks, notebookId, userId, &notebook)
	return notebook
}
func (this *NotebookService) GetNotebookById(notebookId string) info.Notebook {
	notebook := info.Notebook{}
	db.Get(db.Notebooks, notebookId, &notebook)
	return notebook
}

// 得到用户下所有的notebook
// 排序好之后返回
// [ok]
func (this *NotebookService) GetNotebooks(userId string) info.SubNotebooks {
	userNotebooks := []info.Notebook{}
	db.Notebooks.Find(bson.M{"UserId": bson.ObjectIdHex(userId)}).All(&userNotebooks)

	if len(userNotebooks) == 0 {
		return nil
	}
	
	return ParseAndSortNotebooks(userNotebooks, true, true)
}

// share调用, 不需要删除没有父的notebook
// 不需要排序, 因为会重新排序
// 通过notebookIds得到notebooks, 并转成层次有序
func (this *NotebookService) GetNotebooksByNotebookIds(notebookIds []bson.ObjectId) info.SubNotebooks {
	userNotebooks := []info.Notebook{}
	db.Notebooks.Find(bson.M{"_id": bson.M{"$in": notebookIds}}).All(&userNotebooks)
	
	if len(userNotebooks) == 0 {
		return nil
	}
	
	return ParseAndSortNotebooks(userNotebooks, false, false)
}

// 添加
// [ok]
func (this *NotebookService) AddNotebook(notebook info.Notebook) bool {
	err := db.Notebooks.Insert(notebook)
	if err != nil {
		panic(err)
	} else {
	}
	return true
}

// 判断是否是blog
func (this *NotebookService) IsBlog(notebookId string) bool {
	notebook := info.Notebook{}
	db.GetByQWithFields(db.Notebooks, bson.M{"_id": bson.ObjectIdHex(notebookId)}, []string{"IsBlog"}, &notebook);
	return notebook.IsBlog
}

// 判断是否是我的notebook
func (this *NotebookService) IsMyNotebook(notebookId, userId string) bool {
	return db.Has(db.Notebooks, bson.M{"_id": bson.ObjectIdHex(notebookId), "UserId": bson.ObjectIdHex(userId)})
}

// 更新笔记本信息
// 太广, 不用
/*
func (this *NotebookService) UpdateNotebook(notebook info.Notebook) bool {
	return db.UpdateByIdAndUserId2(db.Notebooks, notebook.NotebookId, notebook.UserId, notebook)
}
*/

// 更新笔记本标题
// [ok]
func (this *NotebookService) UpdateNotebookTitle(notebookId, userId, title string) bool {
	return db.UpdateByIdAndUserIdField(db.Notebooks, notebookId, userId, "Title", title)
}

// 更新notebook
func (this *NotebookService) UpdateNotebook(userId, notebookId string, needUpdate bson.M) bool {
	needUpdate["UpdatedTime"] = time.Now();
	
	// 如果有IsBlog之类的, 需要特殊处理
	if isBlog, ok := needUpdate["IsBlog"]; ok {
		// 设为blog/取消, 把它下面所有的note都设为isBlog
		if is, ok2 := isBlog.(bool); ok2 {
			q := bson.M{"UserId": bson.ObjectIdHex(userId), 
				"NotebookId": bson.ObjectIdHex(notebookId)}
			data := bson.M{"IsBlog": is}
			if is {
				data["PublicTime"] = time.Now()
			}
			db.UpdateByQMap(db.Notes, q, data)
				
			// noteContents也更新, 这个就麻烦了, noteContents表没有NotebookId
			// 先查该notebook下所有notes, 得到id
			notes := []info.Note{}
			db.ListByQWithFields(db.Notes, q, []string{"_id"}, &notes)
			if len(notes) > 0 {
				noteIds := make([]bson.ObjectId, len(notes))
				for i, each := range notes {
					noteIds[i] = each.NoteId
				}
				db.UpdateByQMap(db.NoteContents, bson.M{"_id": bson.M{"$in": noteIds}}, bson.M{"IsBlog": isBlog})
			}
		}
	}
	
	return db.UpdateByIdAndUserIdMap(db.Notebooks, notebookId, userId, needUpdate)
}

// 查看是否有子notebook
// 先查看该notebookId下是否有notes, 没有则删除
func (this *NotebookService) DeleteNotebook(userId, notebookId string) (bool, string) {
	if db.Count(db.Notebooks, bson.M{"ParentNotebookId": bson.ObjectIdHex(notebookId), 
		"UserId": bson.ObjectIdHex(userId)}) == 0 { // 无
		if db.Count(db.Notes, bson.M{"NotebookId": bson.ObjectIdHex(notebookId), 
			"UserId": bson.ObjectIdHex(userId),
			"IsTrash": false}) == 0 { // 不包含trash
			return db.DeleteByIdAndUserId(db.Notebooks, notebookId, userId), ""
		}
		return false, "笔记本下有笔记"
	} else {
		return false, "笔记本下有子笔记本"
	}
}

// 排序
// 传入 notebookId => Seq
// 为什么要传入userId, 防止修改其它用户的信息 (恶意)
// [ok]
func (this *NotebookService) SortNotebooks(userId string, notebookId2Seqs map[string]int) bool {
	if len(notebookId2Seqs) == 0 {
		return false
	}

	for notebookId, seq := range notebookId2Seqs {
		if !db.UpdateByIdAndUserIdField2(db.Notebooks, bson.ObjectIdHex(notebookId), bson.ObjectIdHex(userId), "Seq", seq) {
			return false
		}
	}
	
	return true
}

func (this *NotebookService) DragNotebooks(userId string, curNotebookId string, parentNotebookId string, siblings []string) bool {
	userIdO := bson.ObjectIdHex(userId)
	
	ok := false
	// 如果没parentNotebookId, 则parentNotebookId设空
	if(parentNotebookId == "") {
		ok = db.UpdateByIdAndUserIdField2(db.Notebooks, bson.ObjectIdHex(curNotebookId), userIdO, "ParentNotebookId", "");
	} else {
		ok = db.UpdateByIdAndUserIdField2(db.Notebooks, bson.ObjectIdHex(curNotebookId), userIdO, "ParentNotebookId", bson.ObjectIdHex(parentNotebookId));
	}
	
	if !ok {
		return false
	}

	// 排序
	for seq, notebookId := range siblings {
		if !db.UpdateByIdAndUserIdField2(db.Notebooks, bson.ObjectIdHex(notebookId), userIdO, "Seq", seq) {
			return false
		}
	}
	
	return true
}

// 重新统计笔记本下的笔记数目
// noteSevice: AddNote, CopyNote, CopySharedNote, MoveNote
// trashService: DeleteNote (recove不用, 都统一在MoveNote里了)
func (this *NotebookService) ReCountNotebookNumberNotes(notebookId string) bool {
	notebookIdO := bson.ObjectIdHex(notebookId)
	count := db.Count(db.Notes, bson.M{"NotebookId": notebookIdO, "IsTrash": false})
	Log(count)
	Log(notebookId)
	return db.UpdateByQField(db.Notebooks, bson.M{"_id": notebookIdO}, "NumberNotes", count)
}

func (this *NotebookService) ReCountAll() {
	/*
	// 得到所有笔记本
	notebooks := []info.Notebook{}
	db.ListByQWithFields(db.Notebooks, bson.M{}, []string{"NotebookId"}, &notebooks)
	
	for _, each := range notebooks {
		this.ReCountNotebookNumberNotes(each.NotebookId.Hex())
	}
	*/
}
