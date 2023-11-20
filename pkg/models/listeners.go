// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public Licensee as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public Licensee for more details.
//
// You should have received a copy of the GNU Affero General Public Licensee
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"encoding/json"
	"strconv"
	"time"

	"code.vikunja.io/api/pkg/config"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/metrics"
	"code.vikunja.io/api/pkg/modules/keyvalue"
	"code.vikunja.io/api/pkg/notifications"
	"code.vikunja.io/api/pkg/user"

	"github.com/ThreeDotsLabs/watermill/message"
	"xorm.io/xorm"
)

// RegisterListeners registers all event listeners
func RegisterListeners() {
	events.RegisterListener((&ProjectCreatedEvent{}).Name(), &IncreaseProjectCounter{})
	events.RegisterListener((&ProjectDeletedEvent{}).Name(), &DecreaseProjectCounter{})
	events.RegisterListener((&TaskCreatedEvent{}).Name(), &IncreaseTaskCounter{})
	events.RegisterListener((&TaskDeletedEvent{}).Name(), &DecreaseTaskCounter{})
	events.RegisterListener((&TeamDeletedEvent{}).Name(), &DecreaseTeamCounter{})
	events.RegisterListener((&TeamCreatedEvent{}).Name(), &IncreaseTeamCounter{})
	events.RegisterListener((&TaskCommentCreatedEvent{}).Name(), &SendTaskCommentNotification{})
	events.RegisterListener((&TaskAssigneeCreatedEvent{}).Name(), &SendTaskAssignedNotification{})
	events.RegisterListener((&TaskDeletedEvent{}).Name(), &SendTaskDeletedNotification{})
	events.RegisterListener((&ProjectCreatedEvent{}).Name(), &SendProjectCreatedNotification{})
	events.RegisterListener((&TaskAssigneeCreatedEvent{}).Name(), &SubscribeAssigneeToTask{})
	events.RegisterListener((&TeamMemberAddedEvent{}).Name(), &SendTeamMemberAddedNotification{})
	events.RegisterListener((&TaskCommentUpdatedEvent{}).Name(), &HandleTaskCommentEditMentions{})
	events.RegisterListener((&TaskCreatedEvent{}).Name(), &HandleTaskCreateMentions{})
	events.RegisterListener((&TaskUpdatedEvent{}).Name(), &HandleTaskUpdatedMentions{})
	events.RegisterListener((&UserDataExportRequestedEvent{}).Name(), &HandleUserDataExport{})
	events.RegisterListener((&TaskCommentCreatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskCommentUpdatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskCommentDeletedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskAssigneeCreatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskAssigneeDeletedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskAttachmentCreatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskAttachmentDeletedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskRelationCreatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskRelationDeletedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	if config.TypesenseEnabled.GetBool() {
		events.RegisterListener((&TaskDeletedEvent{}).Name(), &RemoveTaskFromTypesense{})
		events.RegisterListener((&TaskCreatedEvent{}).Name(), &AddTaskToTypesense{})
	}
	if config.WebhooksEnabled.GetBool() {
		RegisterEventForWebhook(&TaskCreatedEvent{})
		RegisterEventForWebhook(&TaskUpdatedEvent{})
		RegisterEventForWebhook(&TaskDeletedEvent{})
		RegisterEventForWebhook(&TaskAssigneeCreatedEvent{})
		RegisterEventForWebhook(&TaskAssigneeDeletedEvent{})
		RegisterEventForWebhook(&TaskCommentCreatedEvent{})
		RegisterEventForWebhook(&TaskCommentUpdatedEvent{})
		RegisterEventForWebhook(&TaskCommentDeletedEvent{})
		RegisterEventForWebhook(&TaskAttachmentCreatedEvent{})
		RegisterEventForWebhook(&TaskAttachmentDeletedEvent{})
		RegisterEventForWebhook(&TaskRelationCreatedEvent{})
		RegisterEventForWebhook(&TaskRelationDeletedEvent{})
		RegisterEventForWebhook(&ProjectUpdatedEvent{})
		RegisterEventForWebhook(&ProjectDeletedEvent{})
		RegisterEventForWebhook(&ProjectSharedWithUserEvent{})
		RegisterEventForWebhook(&ProjectSharedWithTeamEvent{})
	}
}

//////
// Task Events

// IncreaseTaskCounter  represents a listener
type IncreaseTaskCounter struct {
}

// Name defines the name for the IncreaseTaskCounter listener
func (s *IncreaseTaskCounter) Name() string {
	return "task.counter.increase"
}

// Handle is executed when the event IncreaseTaskCounter listens on is fired
func (s *IncreaseTaskCounter) Handle(_ *message.Message) (err error) {
	return keyvalue.IncrBy(metrics.TaskCountKey, 1)
}

// DecreaseTaskCounter  represents a listener
type DecreaseTaskCounter struct {
}

// Name defines the name for the DecreaseTaskCounter listener
func (s *DecreaseTaskCounter) Name() string {
	return "task.counter.decrease"
}

// Handle is executed when the event DecreaseTaskCounter listens on is fired
func (s *DecreaseTaskCounter) Handle(_ *message.Message) (err error) {
	return keyvalue.DecrBy(metrics.TaskCountKey, 1)
}

func notifyMentionedUsers(sess *xorm.Session, task *Task, text string, n notifications.NotificationWithSubject) (users map[int64]*user.User, err error) {
	users, err = FindMentionedUsersInText(sess, text)
	if err != nil {
		return
	}

	if len(users) == 0 {
		return
	}

	log.Debugf("Processing %d mentioned users for text %d", len(users), n.SubjectID())

	var notified int
	for _, u := range users {
		can, _, err := task.CanRead(sess, u)
		if err != nil {
			return users, err
		}

		if !can {
			continue
		}

		// Don't notify a user if they were already notified
		dbn, err := notifications.GetNotificationsForNameAndUser(sess, u.ID, n.Name(), n.SubjectID())
		if err != nil {
			return users, err
		}

		if len(dbn) > 0 {
			continue
		}

		err = notifications.Notify(u, n)
		if err != nil {
			return users, err
		}
		notified++
	}

	log.Debugf("Notified %d mentioned users for text %d", notified, n.SubjectID())

	return
}

// SendTaskCommentNotification  represents a listener
type SendTaskCommentNotification struct {
}

// Name defines the name for the SendTaskCommentNotification listener
func (s *SendTaskCommentNotification) Name() string {
	return "task.comment.notification.send"
}

// Handle is executed when the event SendTaskCommentNotification listens on is fired
func (s *SendTaskCommentNotification) Handle(msg *message.Message) (err error) {
	event := &TaskCommentCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	n := &TaskCommentNotification{
		Doer:      event.Doer,
		Task:      event.Task,
		Comment:   event.Comment,
		Mentioned: true,
	}
	mentionedUsers, err := notifyMentionedUsers(sess, event.Task, event.Comment.Comment, n)
	if err != nil {
		return err
	}

	subscribers, err := getSubscribersForEntity(sess, SubscriptionEntityTask, event.Task.ID)
	if err != nil {
		return err
	}

	log.Debugf("Sending task comment notifications to %d subscribers for task %d", len(subscribers), event.Task.ID)

	for _, subscriber := range subscribers {
		if subscriber.UserID == event.Doer.ID {
			continue
		}

		if _, has := mentionedUsers[subscriber.UserID]; has {
			continue
		}

		n := &TaskCommentNotification{
			Doer:    event.Doer,
			Task:    event.Task,
			Comment: event.Comment,
		}
		err = notifications.Notify(subscriber.User, n)
		if err != nil {
			return
		}
	}

	return
}

// HandleTaskCommentEditMentions  represents a listener
type HandleTaskCommentEditMentions struct {
}

// Name defines the name for the HandleTaskCommentEditMentions listener
func (s *HandleTaskCommentEditMentions) Name() string {
	return "handle.task.comment.edit.mentions"
}

// Handle is executed when the event HandleTaskCommentEditMentions listens on is fired
func (s *HandleTaskCommentEditMentions) Handle(msg *message.Message) (err error) {
	event := &TaskCommentUpdatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	n := &TaskCommentNotification{
		Doer:      event.Doer,
		Task:      event.Task,
		Comment:   event.Comment,
		Mentioned: true,
	}
	_, err = notifyMentionedUsers(sess, event.Task, event.Comment.Comment, n)
	return err
}

// SendTaskAssignedNotification  represents a listener
type SendTaskAssignedNotification struct {
}

// Name defines the name for the SendTaskAssignedNotification listener
func (s *SendTaskAssignedNotification) Name() string {
	return "task.assigned.notification.send"
}

// Handle is executed when the event SendTaskAssignedNotification listens on is fired
func (s *SendTaskAssignedNotification) Handle(msg *message.Message) (err error) {
	event := &TaskAssigneeCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	subscribers, err := getSubscribersForEntity(sess, SubscriptionEntityTask, event.Task.ID)
	if err != nil {
		return err
	}

	log.Debugf("Sending task assigned notifications to %d subscribers for task %d", len(subscribers), event.Task.ID)

	task, err := GetTaskByIDSimple(sess, event.Task.ID)
	if err != nil {
		return err
	}

	for _, subscriber := range subscribers {
		if subscriber.UserID == event.Doer.ID {
			continue
		}

		n := &TaskAssignedNotification{
			Doer:     event.Doer,
			Task:     &task,
			Assignee: event.Assignee,
		}
		err = notifications.Notify(subscriber.User, n)
		if err != nil {
			return
		}
	}

	return nil
}

// SendTaskDeletedNotification  represents a listener
type SendTaskDeletedNotification struct {
}

// Name defines the name for the SendTaskDeletedNotification listener
func (s *SendTaskDeletedNotification) Name() string {
	return "task.deleted.notification.send"
}

// Handle is executed when the event SendTaskDeletedNotification listens on is fired
func (s *SendTaskDeletedNotification) Handle(msg *message.Message) (err error) {
	event := &TaskDeletedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	var subscribers []*Subscription
	subscribers, err = getSubscribersForEntity(sess, SubscriptionEntityTask, event.Task.ID)
	// If the task does not exist and no one has explicitly subscribed to it, we won't find any subscriptions for it.
	// Hence, we need to check for subscriptions to the parent project manually.
	if err != nil && (IsErrTaskDoesNotExist(err) || IsErrProjectDoesNotExist(err)) {
		subscribers, err = getSubscribersForEntity(sess, SubscriptionEntityProject, event.Task.ProjectID)
	}
	if err != nil {
		return err
	}

	log.Debugf("Sending task deleted notifications to %d subscribers for task %d", len(subscribers), event.Task.ID)

	for _, subscriber := range subscribers {
		if subscriber.UserID == event.Doer.ID {
			continue
		}

		n := &TaskDeletedNotification{
			Doer: event.Doer,
			Task: event.Task,
		}
		err = notifications.Notify(subscriber.User, n)
		if err != nil {
			return
		}
	}

	return nil
}

type SubscribeAssigneeToTask struct {
}

// Name defines the name for the SubscribeAssigneeToTask listener
func (s *SubscribeAssigneeToTask) Name() string {
	return "task.assignee.subscribe"
}

// Handle is executed when the event SubscribeAssigneeToTask listens on is fired
func (s *SubscribeAssigneeToTask) Handle(msg *message.Message) (err error) {
	event := &TaskAssigneeCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sub := &Subscription{
		UserID:     event.Assignee.ID,
		EntityType: SubscriptionEntityTask,
		EntityID:   event.Task.ID,
	}

	sess := db.NewSession()
	defer sess.Close()

	err = sub.Create(sess, event.Assignee)
	if err != nil && !IsErrSubscriptionAlreadyExists(err) {
		return err
	}

	return sess.Commit()
}

// HandleTaskCreateMentions  represents a listener
type HandleTaskCreateMentions struct {
}

// Name defines the name for the HandleTaskCreateMentions listener
func (s *HandleTaskCreateMentions) Name() string {
	return "task.created.mentions"
}

// Handle is executed when the event HandleTaskCreateMentions listens on is fired
func (s *HandleTaskCreateMentions) Handle(msg *message.Message) (err error) {
	event := &TaskCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	n := &UserMentionedInTaskNotification{
		Task:  event.Task,
		Doer:  event.Doer,
		IsNew: true,
	}
	_, err = notifyMentionedUsers(sess, event.Task, event.Task.Description, n)
	return err
}

// HandleTaskUpdatedMentions  represents a listener
type HandleTaskUpdatedMentions struct {
}

// Name defines the name for the HandleTaskUpdatedMentions listener
func (s *HandleTaskUpdatedMentions) Name() string {
	return "task.updated.mentions"
}

// Handle is executed when the event HandleTaskUpdatedMentions listens on is fired
func (s *HandleTaskUpdatedMentions) Handle(msg *message.Message) (err error) {
	event := &TaskUpdatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	n := &UserMentionedInTaskNotification{
		Task:  event.Task,
		Doer:  event.Doer,
		IsNew: false,
	}

	_, err = notifyMentionedUsers(sess, event.Task, event.Task.Description, n)
	return err
}

// HandleTaskUpdateLastUpdated  represents a listener
type HandleTaskUpdateLastUpdated struct {
}

// Name defines the name for the HandleTaskUpdateLastUpdated listener
func (s *HandleTaskUpdateLastUpdated) Name() string {
	return "handle.task.update.last.updated"
}

// Handle is executed when the event HandleTaskUpdateLastUpdated listens on is fired
func (s *HandleTaskUpdateLastUpdated) Handle(msg *message.Message) (err error) {
	// Using a map here allows us to plug this listener to all kinds of task events
	event := map[string]interface{}{}
	err = json.Unmarshal(msg.Payload, &event)
	if err != nil {
		return err
	}

	task, is := event["Task"].(map[string]interface{})
	if !is {
		log.Errorf("Event payload does not contain task ID")
		return
	}

	taskID, is := task["id"]
	if !is {
		log.Errorf("Event payload does not contain a valid task ID")
		return
	}

	var taskIDInt int64
	switch v := taskID.(type) {
	case int64:
		taskIDInt = v
	case int:
		taskIDInt = int64(v)
	case int32:
		taskIDInt = int64(v)
	case float64:
		taskIDInt = int64(v)
	case float32:
		taskIDInt = int64(v)
	default:
		log.Errorf("Event payload does not contain a valid task ID")
		return
	}

	sess := db.NewSession()
	defer sess.Close()

	return updateTaskLastUpdated(sess, &Task{ID: taskIDInt})
}

// RemoveTaskFromTypesense represents a listener
type RemoveTaskFromTypesense struct {
}

// Name defines the name for the RemoveTaskFromTypesense listener
func (s *RemoveTaskFromTypesense) Name() string {
	return "remove.task.from.typesense"
}

// Handle is executed when the event RemoveTaskFromTypesense listens on is fired
func (s *RemoveTaskFromTypesense) Handle(msg *message.Message) (err error) {
	event := &TaskDeletedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	log.Debugf("[Typesense Sync] Removing task %d from Typesense", event.Task.ID)

	_, err = typesenseClient.
		Collection("tasks").
		Document(strconv.FormatInt(event.Task.ID, 10)).
		Delete()
	return err
}

// AddTaskToTypesense  represents a listener
type AddTaskToTypesense struct {
}

// Name defines the name for the AddTaskToTypesense listener
func (l *AddTaskToTypesense) Name() string {
	return "add.task.to.typesense"
}

// Handle is executed when the event AddTaskToTypesense listens on is fired
func (l *AddTaskToTypesense) Handle(msg *message.Message) (err error) {
	event := &TaskCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	log.Debugf("New task %d created, adding to typesense…", event.Task.ID)

	s := db.NewSession()
	defer s.Close()
	ttask, err := getTypesenseTaskForTask(s, event.Task, nil)
	if err != nil {
		return err
	}

	_, err = typesenseClient.Collection("tasks").
		Documents().
		Create(ttask)
	return
}

///////
// Project Event Listeners

type IncreaseProjectCounter struct {
}

func (s *IncreaseProjectCounter) Name() string {
	return "project.counter.increase"
}

func (s *IncreaseProjectCounter) Handle(_ *message.Message) (err error) {
	return keyvalue.IncrBy(metrics.ProjectCountKey, 1)
}

type DecreaseProjectCounter struct {
}

func (s *DecreaseProjectCounter) Name() string {
	return "project.counter.decrease"
}

func (s *DecreaseProjectCounter) Handle(_ *message.Message) (err error) {
	return keyvalue.DecrBy(metrics.ProjectCountKey, 1)
}

// SendProjectCreatedNotification  represents a listener
type SendProjectCreatedNotification struct {
}

// Name defines the name for the SendProjectCreatedNotification listener
func (s *SendProjectCreatedNotification) Name() string {
	return "send.project.created.notification"
}

// Handle is executed when the event SendProjectCreatedNotification listens on is fired
func (s *SendProjectCreatedNotification) Handle(msg *message.Message) (err error) {
	event := &ProjectCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	subscribers, err := getSubscribersForEntity(sess, SubscriptionEntityProject, event.Project.ID)
	if err != nil {
		return err
	}

	log.Debugf("Sending project created notifications to %d subscribers for project %d", len(subscribers), event.Project.ID)

	for _, subscriber := range subscribers {
		if subscriber.UserID == event.Doer.ID {
			continue
		}

		n := &ProjectCreatedNotification{
			Doer:    event.Doer,
			Project: event.Project,
		}
		err = notifications.Notify(subscriber.User, n)
		if err != nil {
			return
		}
	}

	return nil
}

// WebhookListener represents a listener
type WebhookListener struct {
	EventName string
}

// Name defines the name for the WebhookListener listener
func (wl *WebhookListener) Name() string {
	return "webhook.listener"
}

type WebhookPayload struct {
	EventName string      `json:"event_name"`
	Time      time.Time   `json:"time"`
	Data      interface{} `json:"data"`
}

func getProjectIDFromAnyEvent(eventPayload map[string]interface{}) int64 {
	if task, has := eventPayload["task"]; has {
		t := task.(map[string]interface{})
		if projectID, has := t["project_id"]; has {
			switch v := projectID.(type) {
			case int64:
				return v
			case float64:
				return int64(v)
			}
			return projectID.(int64)
		}
	}

	if project, has := eventPayload["project"]; has {
		t := project.(map[string]interface{})
		if projectID, has := t["id"]; has {
			switch v := projectID.(type) {
			case int64:
				return v
			case float64:
				return int64(v)
			}
			return projectID.(int64)
		}
	}

	return 0
}

// Handle is executed when the event WebhookListener listens on is fired
func (wl *WebhookListener) Handle(msg *message.Message) (err error) {
	var event map[string]interface{}
	err = json.Unmarshal(msg.Payload, &event)
	if err != nil {
		return err
	}

	projectID := getProjectIDFromAnyEvent(event)
	if projectID == 0 {
		log.Debugf("event %s does not contain a project id, not handling webhook", wl.EventName)
		return nil
	}

	s := db.NewSession()
	defer s.Close()

	ws := []*Webhook{}
	err = s.Where("project_id = ?", projectID).
		Find(&ws)
	if err != nil {
		return err
	}

	var webhook *Webhook
	for _, w := range ws {
		for _, e := range w.Events {
			if e == wl.EventName {
				webhook = w
				break
			}
		}

	}

	if webhook == nil {
		log.Debugf("Did not find any webhook for the %s event for project %d, not sending", wl.EventName, projectID)
		return nil
	}

	err = webhook.sendWebhookPayload(&WebhookPayload{
		EventName: wl.EventName,
		Time:      time.Now(),
		Data:      event,
	})
	return
}

///////
// Team Events

// IncreaseTeamCounter  represents a listener
type IncreaseTeamCounter struct {
}

// Name defines the name for the IncreaseTeamCounter listener
func (s *IncreaseTeamCounter) Name() string {
	return "team.counter.increase"
}

// Handle is executed when the event IncreaseTeamCounter listens on is fired
func (s *IncreaseTeamCounter) Handle(_ *message.Message) (err error) {
	return keyvalue.IncrBy(metrics.TeamCountKey, 1)
}

// DecreaseTeamCounter  represents a listener
type DecreaseTeamCounter struct {
}

// Name defines the name for the DecreaseTeamCounter listener
func (s *DecreaseTeamCounter) Name() string {
	return "team.counter.decrease"
}

// Handle is executed when the event DecreaseTeamCounter listens on is fired
func (s *DecreaseTeamCounter) Handle(_ *message.Message) (err error) {
	return keyvalue.DecrBy(metrics.TeamCountKey, 1)
}

// SendTeamMemberAddedNotification  represents a listener
type SendTeamMemberAddedNotification struct {
}

// Name defines the name for the SendTeamMemberAddedNotification listener
func (s *SendTeamMemberAddedNotification) Name() string {
	return "team.member.added.notification"
}

// Handle is executed when the event SendTeamMemberAddedNotification listens on is fired
func (s *SendTeamMemberAddedNotification) Handle(msg *message.Message) (err error) {
	event := &TeamMemberAddedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	// Don't notify the user themselves
	if event.Doer.ID == event.Member.ID {
		return nil
	}

	return notifications.Notify(event.Member, &TeamMemberAddedNotification{
		Member: event.Member,
		Doer:   event.Doer,
		Team:   event.Team,
	})
}

// HandleUserDataExport  represents a listener
type HandleUserDataExport struct {
}

// Name defines the name for the HandleUserDataExport listener
func (s *HandleUserDataExport) Name() string {
	return "handle.user.data.export"
}

// Handle is executed when the event HandleUserDataExport listens on is fired
func (s *HandleUserDataExport) Handle(msg *message.Message) (err error) {
	event := &UserDataExportRequestedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	log.Debugf("Starting to export user data for user %d...", event.User.ID)

	sess := db.NewSession()
	defer sess.Close()
	err = sess.Begin()
	if err != nil {
		return
	}

	err = ExportUserData(sess, event.User)
	if err != nil {
		_ = sess.Rollback()
		return
	}

	log.Debugf("Done exporting user data for user %d...", event.User.ID)

	err = sess.Commit()
	return err
}
