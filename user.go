package openwechat

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

type User struct {
	Uin               int
	HideInputBarFlag  int
	StarFriend        int
	Sex               int
	AppAccountFlag    int
	VerifyFlag        int
	ContactFlag       int
	WebWxPluginSwitch int
	HeadImgFlag       int
	SnsFlag           int
	UserName          string
	NickName          string
	HeadImgUrl        string
	RemarkName        string
	PYInitial         string
	PYQuanPin         string
	RemarkPYInitial   string
	RemarkPYQuanPin   string
	Signature         string
	MemberCount       int
	MemberList        Members
	OwnerUin          int
	Statues           int
	AttrStatus        int
	Province          string
	City              string
	Alias             string
	UniFriend         int
	DisplayName       string
	ChatRoomId        int
	KeyWord           string
	EncryChatRoomId   string
	IsOwner           int

	Self *Self
}

// implement fmt.Stringer
func (u *User) String() string {
	return fmt.Sprintf("<User:%s>", u.NickName)
}

// 获取用户头像
func (u *User) GetAvatarResponse() (*http.Response, error) {
	return u.Self.Bot.Caller.Client.WebWxGetHeadImg(u.HeadImgUrl)
}

// 下载用户头像
func (u *User) SaveAvatar(filename string) error {
	resp, err := u.GetAvatarResponse()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buffer := bytes.Buffer{}
	if _, err := buffer.ReadFrom(resp.Body); err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(buffer.Bytes())
	return err
}

func (u *User) sendMsg(msg *SendMessage) error {
	msg.FromUserName = u.Self.UserName
	msg.ToUserName = u.UserName
	info := u.Self.Bot.storage.LoginInfo
	request := u.Self.Bot.storage.Request
	return u.Self.Bot.Caller.WebWxSendMsg(msg, info, request)
}

func (u *User) sendText(content string) error {
	msg := NewTextSendMessage(content, u.Self.UserName, u.UserName)
	return u.sendMsg(msg)
}

func (u *User) sendImage(file *os.File) error {
	request := u.Self.Bot.storage.Request
	info := u.Self.Bot.storage.LoginInfo
	return u.Self.Bot.Caller.WebWxSendImageMsg(file, request, info, u.Self.UserName, u.UserName)
}

func (u *User) setRemarkName(remarkName string) error {
	request := u.Self.Bot.storage.Request
	return u.Self.Bot.Caller.WebWxOplog(request, remarkName, u.UserName)
}

// 获取用户的详情
func (u *User) Detail() (*User, error) {
	members := Members{u}
	request := u.Self.Bot.storage.Request
	newMembers, err := u.Self.Bot.Caller.WebWxBatchGetContact(members, request)
	if err != nil {
		return nil, err
	}
	user := newMembers.First()
	user.Self = u.Self
	return user, nil
}

type Self struct {
	*User
	Bot        *Bot
	fileHelper *Friend
	members    Members
	friends    Friends
	groups     Groups
	mps        Mps
}

// 获取所有的好友、群组、公众号信息
func (s *Self) Members(update ...bool) (Members, error) {
	// 首先判断缓存里有没有,如果没有则去更新缓存
	if s.members == nil {
		if err := s.updateMembers(); err != nil {
			return nil, err
		}
		return s.members, nil
	}
	// 判断是否需要更新,如果传入的参数不为nil,则取最后一个
	var isUpdate bool
	if len(update) > 0 {
		isUpdate = update[len(update)-1]
	}
	// 如果需要更新，则直接更新缓存
	if isUpdate {
		if err := s.updateMembers(); err != nil {
			return nil, err
		}
	}
	return s.members, nil
}

// 更新联系人处理
func (s *Self) updateMembers() error {
	info := s.Bot.storage.LoginInfo
	members, err := s.Bot.Caller.WebWxGetContact(info)
	if err != nil {
		return err
	}
	members.SetOwner(s)
	s.members = members
	return nil
}

// 获取文件传输助手对象，封装成Friend返回
func (s *Self) FileHelper() (*Friend, error) {
	// 如果缓存里有，直接返回，否则去联系人里面找
	if s.fileHelper != nil {
		return s.fileHelper, nil
	}
	members, err := s.Members()
	if err != nil {
		return nil, err
	}
	users := members.SearchByUserName(1, "filehelper")
	if users == nil {
		return nil, noSuchUserFoundError
	}
	s.fileHelper = &Friend{users.First()}
	return s.fileHelper, nil
}

// 获取所有的好友
func (s *Self) Friends(update ...bool) (Friends, error) {
	if s.friends == nil {
		if err := s.updateFriends(update...); err != nil {
			return nil, err
		}
	}
	return s.friends, nil
}

// 获取所有的群组
func (s *Self) Groups(update ...bool) (Groups, error) {
	if s.groups == nil {
		if err := s.updateGroups(update...); err != nil {
			return nil, err
		}
	}
	return s.groups, nil
}

// 获取所有的公众号
func (s *Self) Mps(update ...bool) (Mps, error) {
	if s.mps == nil {
		if err := s.updateMps(update...); err != nil {
			return nil, err
		}
	}
	return s.mps, nil
}

// 更新好友处理
func (s *Self) updateFriends(update ...bool) error {
	var isUpdate bool
	if len(update) > 0 {
		isUpdate = update[len(update)-1]
	}
	if isUpdate || s.members == nil {
		if err := s.updateMembers(); err != nil {
			return err
		}
	}
	var friends Friends
	for _, member := range s.members {
		if isFriend(*member) {
			friend := &Friend{member}
			friend.Self = s
			friends = append(friends, friend)
		}
	}
	s.friends = friends
	return nil
}

// 更新群组处理
func (s *Self) updateGroups(update ...bool) error {
	var isUpdate bool
	if len(update) > 0 {
		isUpdate = update[len(update)-1]
	}
	if isUpdate || s.members == nil {
		if err := s.updateMembers(); err != nil {
			return err
		}
	}
	var groups Groups
	for _, member := range s.members {
		if isGroup(*member) {
			group := &Group{member}
			groups = append(groups, group)
		}
	}
	s.groups = groups
	return nil
}

func (s *Self) updateMps(update ...bool) error {
	var isUpdate bool
	if len(update) > 0 {
		isUpdate = update[len(update)-1]
	}
	if isUpdate || s.members == nil {
		if err := s.updateMembers(); err != nil {
			return err
		}
	}
	var mps Mps
	for _, member := range s.members {
		if isMP(*member) {
			mp := &Mp{member}
			mps = append(mps, mp)
		}
	}
	s.mps = mps
	return nil
}

// 更新所有的联系人信息
func (s *Self) UpdateMembersDetail() error {
	// 先获取所有的联系人
	members, err := s.Members()
	if err != nil {
		return err
	}
	// 获取他们的数量
	count := members.Count()
	// 一次更新50个,分情况讨论

	// 获取总的需要更新的次数
	var times int
	if count < 50 {
		times = 1
	} else {
		times = count / 50
	}
	var newMembers Members
	request := s.Bot.storage.Request
	var pMembers Members
	// 分情况依次更新
	for i := 1; i <= times; i++ {
		if times == 1 {
			pMembers = members
		} else {
			pMembers = members[(i-1)*50 : i*50]
		}
		nMembers, err := s.Bot.Caller.WebWxBatchGetContact(pMembers, request)
		if err != nil {
			return err
		}
		newMembers = append(newMembers, nMembers...)
	}
	// 最后判断是否全部更新完毕
	total := times * 50
	if total < count {
		// 将全部剩余的更新完毕
		left := count - total
		pMembers = members[total : total+left]
		nMembers, err := s.Bot.Caller.WebWxBatchGetContact(pMembers, request)
		if err != nil {
			return err
		}
		newMembers = append(newMembers, nMembers...)
	}
	if len(newMembers) != 0 {
		newMembers.SetOwner(s)
		s.members = newMembers
	}
	return nil
}

type Members []*User

func (m Members) Count() int {
	return len(m)
}

func (m Members) First() *User {
	if m.Count() > 0 {
		u := m[0]
		return u
	}
	return nil
}

func (m Members) Last() *User {
	if m.Count() > 0 {
		u := m[m.Count()-1]
		return u
	}
	return nil
}

func (m Members) SetOwner(s *Self) {
	for _, member := range m {
		member.Self = s
	}
}

func (m Members) SearchByUserName(limit int, username string) (results Members) {
	return m.Search(limit, func(user *User) bool { return user.UserName == username })
}

func (m Members) SearchByNickName(limit int, nickName string) (results Members) {
	return m.Search(limit, func(user *User) bool { return user.NickName == nickName })
}

func (m Members) SearchByRemarkName(limit int, remarkName string) (results Members) {
	return m.Search(limit, func(user *User) bool { return user.RemarkName == remarkName })
}

func (m Members) Search(limit int, condFuncList ...func(user *User) bool) (results Members) {
	if condFuncList == nil {
		return m
	}
	if limit <= 0 {
		limit = m.Count()
	}
	for _, member := range m {
		if count := len(results); count == limit {
			break
		}
		var passCount int
		for _, condFunc := range condFuncList {
			if condFunc(member) {
				passCount++
			} else {
				break
			}
		}
		if passCount == len(condFuncList) {
			results = append(results, member)
		}
	}
	return
}
