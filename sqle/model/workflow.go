package model

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/actiontech/sqle/sqle/errors"
	"github.com/jinzhu/gorm"
)

type WorkflowTemplate struct {
	Model
	Name                          string
	Desc                          string
	AllowSubmitWhenLessAuditLevel string

	Steps     []*WorkflowStepTemplate `json:"-" gorm:"foreignkey:workflowTemplateId"`
	Instances []*Instance             `gorm:"foreignkey:WorkflowTemplateId"`
}

const (
	WorkflowStepTypeSQLReview      = "sql_review"
	WorkflowStepTypeSQLExecute     = "sql_execute"
	WorkflowStepTypeCreateWorkflow = "create_workflow"
	WorkflowStepTypeUpdateWorkflow = "update_workflow"
)

type WorkflowStepTemplate struct {
	Model
	Number               uint   `gorm:"index; column:step_number"`
	WorkflowTemplateId   int    `gorm:"index"`
	Typ                  string `gorm:"column:type; not null"`
	Desc                 string
	ApprovedByAuthorized sql.NullBool `gorm:"column:approved_by_authorized"`

	Users []*User `gorm:"many2many:workflow_step_template_user"`
}

func (s *Storage) GetWorkflowTemplateByName(name string) (*WorkflowTemplate, bool, error) {
	workflowTemplate := &WorkflowTemplate{}
	err := s.db.Where("name = ?", name).First(workflowTemplate).Error
	if err == gorm.ErrRecordNotFound {
		return workflowTemplate, false, nil
	}
	return workflowTemplate, true, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetWorkflowTemplateById(id uint) (*WorkflowTemplate, bool, error) {
	workflowTemplate := &WorkflowTemplate{}
	err := s.db.Where("id = ?", id).First(workflowTemplate).Error
	if err == gorm.ErrRecordNotFound {
		return workflowTemplate, false, nil
	}
	return workflowTemplate, true, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetWorkflowStepsByTemplateId(id uint) ([]*WorkflowStepTemplate, error) {
	steps := []*WorkflowStepTemplate{}
	err := s.db.Preload("Users").Where("workflow_template_id = ?", id).Find(&steps).Error
	return steps, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetWorkflowStepsDetailByTemplateId(id uint) ([]*WorkflowStepTemplate, error) {
	steps := []*WorkflowStepTemplate{}
	err := s.db.Preload("Users").Where("workflow_template_id = ?", id).Find(&steps).Error
	return steps, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) SaveWorkflowTemplate(template *WorkflowTemplate) error {
	return s.TxExec(func(tx *sql.Tx) error {
		result, err := tx.Exec("INSERT INTO workflow_templates (name, `desc`, `allow_submit_when_less_audit_level`) values (?, ?, ?)",
			template.Name, template.Desc, template.AllowSubmitWhenLessAuditLevel)
		if err != nil {
			return err
		}
		templateId, err := result.LastInsertId()
		if err != nil {
			return err
		}
		template.ID = uint(templateId)
		for _, step := range template.Steps {
			result, err = tx.Exec("INSERT INTO workflow_step_templates (step_number, workflow_template_id, type, `desc`, approved_by_authorized) values (?,?,?,?,?)",
				step.Number, templateId, step.Typ, step.Desc, step.ApprovedByAuthorized)
			if err != nil {
				return err
			}
			stepId, err := result.LastInsertId()
			if err != nil {
				return err
			}
			step.ID = uint(stepId)
			for _, user := range step.Users {
				_, err = tx.Exec("INSERT INTO workflow_step_template_user (workflow_step_template_id, user_id) values (?,?)",
					stepId, user.ID)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *Storage) UpdateWorkflowTemplateSteps(templateId uint, steps []*WorkflowStepTemplate) error {
	return s.TxExec(func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE workflow_step_templates SET workflow_template_id = NULL WHERE workflow_template_id = ?",
			templateId)
		if err != nil {
			return err
		}
		for _, step := range steps {
			result, err := tx.Exec("INSERT INTO workflow_step_templates (step_number, workflow_template_id, type, `desc`, approved_by_authorized) values (?,?,?,?,?)",
				step.Number, templateId, step.Typ, step.Desc, step.ApprovedByAuthorized)
			if err != nil {
				return err
			}
			stepId, err := result.LastInsertId()
			if err != nil {
				return err
			}
			step.ID = uint(stepId)
			for _, user := range step.Users {
				_, err = tx.Exec("INSERT INTO workflow_step_template_user (workflow_step_template_id, user_id) values (?,?)",
					stepId, user.ID)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *Storage) UpdateWorkflowTemplateInstances(workflowTemplate *WorkflowTemplate,
	instances ...*Instance) error {
	err := s.db.Model(workflowTemplate).Association("Instances").Replace(instances).Error
	return errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetWorkflowTemplateTip() ([]*WorkflowTemplate, error) {
	templates := []*WorkflowTemplate{}
	err := s.db.Select("name").Find(&templates).Error
	return templates, errors.New(errors.ConnectStorageError, err)
}

type Workflow struct {
	Model
	Subject          string
	Desc             string
	CreateUserId     uint
	WorkflowRecordId uint

	CreateUser    *User             `gorm:"foreignkey:CreateUserId"`
	Record        *WorkflowRecord   `gorm:"foreignkey:WorkflowRecordId"`
	RecordHistory []*WorkflowRecord `gorm:"many2many:workflow_record_history;"`
}

const (
	WorkflowStatusRunning       = "on_process"
	WorkflowStatusReject        = "rejected"
	WorkflowStatusCancel        = "canceled"
	WorkflowStatusExecScheduled = "exec_scheduled"
	WorkflowStatusExecuting     = "executing"
	WorkflowStatusExecFailed    = "exec_failed"
	WorkflowStatusFinish        = "finished"
)

type WorkflowRecord struct {
	Model
	TaskId                uint `gorm:"index"`
	CurrentWorkflowStepId uint
	Status                string `gorm:"default:\"on_process\""`
	ScheduledAt           *time.Time
	ScheduleUserId        uint

	CurrentStep *WorkflowStep   `gorm:"foreignkey:CurrentWorkflowStepId"`
	Steps       []*WorkflowStep `gorm:"foreignkey:WorkflowRecordId"`
}

const (
	WorkflowStepStateInit    = "initialized"
	WorkflowStepStateApprove = "approved"
	WorkflowStepStateReject  = "rejected"
)

type WorkflowStep struct {
	Model
	OperationUserId        uint
	OperateAt              *time.Time
	WorkflowId             uint   `gorm:"index; not null"`
	WorkflowRecordId       uint   `gorm:"index; not null"`
	WorkflowStepTemplateId uint   `gorm:"index; not null"`
	State                  string `gorm:"default:\"initialized\""`
	Reason                 string

	Assignees     []*User               `gorm:"many2many:workflow_step_user"`
	Template      *WorkflowStepTemplate `gorm:"foreignkey:WorkflowStepTemplateId"`
	OperationUser *User                 `gorm:"foreignkey:OperationUserId"`
}

func generateWorkflowStepByTemplate(stepsTemplate []*WorkflowStepTemplate, allInspector []*User) []*WorkflowStep {
	steps := make([]*WorkflowStep, 0, len(stepsTemplate))
	for _, st := range stepsTemplate {
		step := &WorkflowStep{
			WorkflowStepTemplateId: st.ID,
			Assignees:              st.Users,
		}
		if st.ApprovedByAuthorized.Bool {
			step.Assignees = allInspector
		}
		steps = append(steps, step)
	}
	return steps
}

func (w *Workflow) cloneWorkflowStep() []*WorkflowStep {
	steps := make([]*WorkflowStep, 0, len(w.Record.Steps))
	for _, step := range w.Record.Steps {
		steps = append(steps, &WorkflowStep{
			WorkflowStepTemplateId: step.Template.ID,
			WorkflowId:             w.ID,
			Assignees:              step.Assignees,
		})
	}
	return steps
}

func (w *Workflow) CreateUserName() string {
	if w.CreateUser != nil {
		return w.CreateUser.Name
	}
	return ""
}

func (w *Workflow) CurrentStep() *WorkflowStep {
	return w.Record.CurrentStep
}

func (w *Workflow) CurrentAssigneeUser() []*User {
	currentStep := w.CurrentStep()
	if currentStep == nil {
		return []*User{}
	}
	return currentStep.Assignees
}

func (w *Workflow) NextStep() *WorkflowStep {
	var nextIndex int
	for i, step := range w.Record.Steps {
		if step.ID == w.Record.CurrentWorkflowStepId {
			nextIndex = i + 1
			break
		}
	}
	if nextIndex <= len(w.Record.Steps)-1 {
		return w.Record.Steps[nextIndex]
	}
	return nil
}

func (w *Workflow) FinalStep() *WorkflowStep {
	return w.Record.Steps[len(w.Record.Steps)-1]
}

func (w *Workflow) IsOperationUser(user *User) bool {
	if w.CurrentStep() == nil {
		return false
	}
	for _, assUser := range w.CurrentStep().Assignees {
		if user.ID == assUser.ID {
			return true
		}
	}
	return false
}

// IsFirstRecord check the record is the first record in workflow;
// you must load record history first and then use it.
func (w *Workflow) IsFirstRecord(record *WorkflowRecord) bool {
	records := []*WorkflowRecord{}
	records = append(records, w.RecordHistory...)
	records = append(records, w.Record)
	if len(records) > 0 {
		return record == records[0]
	}
	return false
}

func (s *Storage) CreateWorkflow(subject, desc string, user *User, task *Task,
	stepTemplates []*WorkflowStepTemplate) error {

	workflow := &Workflow{
		Subject:      subject,
		Desc:         desc,
		CreateUserId: user.ID,
	}
	record := &WorkflowRecord{
		TaskId: task.ID,
	}

	inspector, err := s.GetUsersByOperationCode(task.Instance, OP_WORKFLOW_AUDIT)
	if err != nil {
		return err
	}

	steps := generateWorkflowStepByTemplate(stepTemplates, inspector)

	tx := s.db.Begin()

	err = tx.Save(record).Error
	if err != nil {
		tx.Rollback()
		return errors.New(errors.ConnectStorageError, err)
	}

	workflow.WorkflowRecordId = record.ID
	err = tx.Save(workflow).Error
	if err != nil {
		tx.Rollback()
		return errors.New(errors.ConnectStorageError, err)
	}

	for _, step := range steps {
		currentStep := step
		currentStep.WorkflowRecordId = record.ID
		currentStep.WorkflowId = workflow.ID
		users := currentStep.Assignees
		currentStep.Assignees = nil
		err = tx.Save(currentStep).Error
		if err != nil {
			tx.Rollback()
			return errors.New(errors.ConnectStorageError, err)
		}
		err = tx.Model(currentStep).Association("Assignees").Replace(users).Error
		if err != nil {
			tx.Rollback()
			return errors.New(errors.ConnectStorageError, err)
		}
	}
	if len(steps) > 0 {
		err = tx.Model(record).Update("current_workflow_step_id", steps[0].ID).Error
		if err != nil {
			tx.Rollback()
			return errors.New(errors.ConnectStorageError, err)
		}
	}
	return errors.New(errors.ConnectStorageError, tx.Commit().Error)
}

func (s *Storage) UpdateWorkflowRecord(w *Workflow, task *Task) error {
	record := &WorkflowRecord{
		TaskId: task.ID,
	}
	steps := w.cloneWorkflowStep()

	tx := s.db.Begin()
	err := tx.Save(record).Error
	if err != nil {
		tx.Rollback()
		return errors.New(errors.ConnectStorageError, err)
	}

	for _, step := range steps {
		currentStep := step
		currentStep.WorkflowRecordId = record.ID
		users := currentStep.Assignees
		currentStep.Assignees = nil
		err = tx.Save(currentStep).Error
		if err != nil {
			tx.Rollback()
			return errors.New(errors.ConnectStorageError, err)
		}
		err = tx.Model(currentStep).Association("Assignees").Replace(users).Error
		if err != nil {
			tx.Rollback()
			return errors.New(errors.ConnectStorageError, err)
		}
	}
	if len(steps) > 0 {
		err = tx.Model(record).Update("current_workflow_step_id", steps[0].ID).Error
		if err != nil {
			tx.Rollback()
			return errors.New(errors.ConnectStorageError, err)
		}
	}
	// update record history
	err = tx.Exec("INSERT INTO workflow_record_history (workflow_record_id, workflow_id) value (?, ?)",
		w.Record.ID, w.ID).Error
	if err != nil {
		tx.Rollback()
		return errors.New(errors.ConnectStorageError, err)
	}

	// update workflow record to new
	if err := tx.Model(&Workflow{}).Where("id = ?", w.ID).
		Update("workflow_record_id", record.ID).Error; err != nil {
		tx.Rollback()
		return errors.New(errors.ConnectStorageError, err)
	}

	return errors.New(errors.ConnectStorageError, tx.Commit().Error)
}

func (s *Storage) UpdateWorkflowStatus(w *Workflow, operateStep *WorkflowStep) error {
	return s.TxExec(func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE workflow_records SET status = ?, current_workflow_step_id = ? WHERE id = ?",
			w.Record.Status, w.Record.CurrentWorkflowStepId, w.Record.ID)
		if err != nil {
			return err
		}
		if operateStep == nil {
			return nil
		}
		_, err = tx.Exec("UPDATE workflow_steps SET operation_user_id = ?, operate_at = ?, state = ?, reason = ? WHERE id = ?",
			operateStep.OperationUserId, operateStep.OperateAt, operateStep.State, operateStep.Reason, operateStep.ID)
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *Storage) UpdateWorkflowSchedule(w *Workflow, userId uint, scheduleTime *time.Time) error {
	err := s.db.Model(&WorkflowRecord{}).Where("id = ?", w.Record.ID).Update(map[string]interface{}{
		"scheduled_at":     scheduleTime,
		"schedule_user_id": userId,
	}).Error
	return errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) getWorkflowStepsByRecordIds(ids []uint) ([]*WorkflowStep, error) {
	steps := []*WorkflowStep{}
	err := s.db.Where("workflow_record_id in (?)", ids).
		Preload("Assignees").
		Preload("OperationUser").Find(&steps).Error
	if err != nil {
		return nil, errors.New(errors.ConnectStorageError, err)
	}
	stepTemplateIds := make([]uint, 0, len(steps))
	for _, step := range steps {
		stepTemplateIds = append(stepTemplateIds, step.WorkflowStepTemplateId)
	}
	stepTemplates := []*WorkflowStepTemplate{}
	err = s.db.Where("id in (?)", stepTemplateIds).Find(&stepTemplates).Error
	if err != nil {
		return nil, errors.New(errors.ConnectStorageError, err)
	}
	for _, step := range steps {
		for _, stepTemplate := range stepTemplates {
			if step.WorkflowStepTemplateId == stepTemplate.ID {
				step.Template = stepTemplate
			}
		}
	}
	return steps, nil
}

func (s *Storage) GetWorkflowDetailById(id string) (*Workflow, bool, error) {
	workflow := &Workflow{}
	err := s.db.Preload("CreateUser", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Record").
		Where("id = ?", id).First(workflow).Error
	if err == gorm.ErrRecordNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.New(errors.ConnectStorageError, err)
	}
	if workflow.Record == nil {
		return nil, false, errors.New(errors.DataConflict, fmt.Errorf("workflow record not exist"))
	}
	steps, err := s.getWorkflowStepsByRecordIds([]uint{workflow.Record.ID})
	if err != nil {
		return nil, false, errors.New(errors.ConnectStorageError, err)
	}
	workflow.Record.Steps = steps
	for _, step := range steps {
		if step.ID == workflow.Record.CurrentWorkflowStepId {
			workflow.Record.CurrentStep = step
		}
	}
	return workflow, true, nil
}

func (s *Storage) GetWorkflowHistoryById(id string) ([]*WorkflowRecord, error) {
	records := []*WorkflowRecord{}
	err := s.db.Model(&WorkflowRecord{}).Select("workflow_records.*").
		Joins("JOIN workflow_record_history AS wrh ON workflow_records.id = wrh.workflow_record_id").
		Where("wrh.workflow_id = ?", id).Scan(&records).Error
	if err != nil {
		return nil, errors.New(errors.ConnectStorageError, err)
	}
	if len(records) == 0 {
		return records, nil
	}
	recordIds := make([]uint, 0, len(records))
	for _, record := range records {
		recordIds = append(recordIds, record.ID)
	}
	steps, err := s.getWorkflowStepsByRecordIds(recordIds)
	if err != nil {
		return nil, errors.New(errors.ConnectStorageError, err)
	}
	for _, record := range records {
		record.Steps = []*WorkflowStep{}
		for _, step := range steps {
			if step.WorkflowRecordId == record.ID && step.State != WorkflowStepStateInit {
				record.Steps = append(record.Steps, step)
			}
		}
	}
	return records, nil
}

// TODO: args `id` using uint
func (s *Storage) GetWorkflowRecordByTaskId(id string) (*WorkflowRecord, bool, error) {
	record := &WorkflowRecord{}
	err := s.db.Model(&WorkflowRecord{}).Select("workflow_records.id").
		Where("workflow_records.task_id = ?", id).Scan(record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.New(errors.ConnectStorageError, err)
	}
	return record, true, nil
}

func (s *Storage) GetWorkflowByTaskId(id uint) (*Workflow, bool, error) {
	workflow := &Workflow{}
	err := s.db.Model(&Workflow{}).Select("workflows.id").
		Joins("LEFT JOIN workflow_records AS wr ON "+
			"workflows.workflow_record_id = wr.id").
		Joins("LEFT JOIN workflow_record_history ON "+
			"workflows.id = workflow_record_history.workflow_id").
		Joins("LEFT JOIN workflow_records AS h_wr ON "+
			"workflow_record_history.workflow_record_id = h_wr.id").
		Where("wr.task_id = ? OR h_wr.task_id = ? AND workflows.id IS NOT NULL", id, id).
		Limit(1).Group("workflows.id").Scan(workflow).Error
	if err == gorm.ErrRecordNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.New(errors.ConnectStorageError, err)
	}
	return workflow, true, nil
}

func (s *Storage) GetLastWorkflow() (*Workflow, bool, error) {
	workflow := new(Workflow)
	err := s.db.Last(workflow).Error
	if err == gorm.ErrRecordNotFound {
		return nil, false, nil
	}
	return workflow, true, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) DeleteWorkflow(workflow *Workflow) error {
	return s.TxExec(func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM workflows WHERE id = ?", workflow.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM workflow_records WHERE id = ?", workflow.WorkflowRecordId)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM workflow_steps WHERE workflow_id = ?", workflow.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM workflow_record_history WHERE workflow_id = ?", workflow.ID)
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *Storage) GetExpiredWorkflows(start time.Time) ([]*Workflow, error) {
	workflows := []*Workflow{}
	err := s.db.Model(&Workflow{}).Select("workflows.id, workflows.workflow_record_id").
		Joins("LEFT JOIN workflow_records ON workflows.workflow_record_id = workflow_records.id").
		Where("workflows.created_at < ? "+
			"AND (workflow_records.status = 'finished' "+
			"OR workflow_records.status = 'canceled' "+
			"OR workflow_records.status IS NULL)", start).
		Scan(&workflows).Error
	return workflows, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetNeedScheduledWorkflows() ([]*Workflow, error) {
	workflows := []*Workflow{}
	err := s.db.Model(&Workflow{}).Select("workflows.id, workflows.workflow_record_id").
		Joins("LEFT JOIN workflow_records ON workflows.workflow_record_id = workflow_records.id").
		Where("workflow_records.scheduled_at IS NOT NULL "+
			"AND workflow_records.scheduled_at <= ? "+
			"AND workflow_records.status = 'on_process'", time.Now()).
		Scan(&workflows).Error
	return workflows, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetWorkflowBySubject(subject string) (*Workflow, bool, error) {
	workflow := &Workflow{Subject: subject}
	err := s.db.Where(*workflow).First(workflow).Error
	if err == gorm.ErrRecordNotFound {
		return workflow, false, nil
	}
	return workflow, true, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) TaskWorkflowIsRunning(taskIds []uint) (bool, error) {
	var workflowRecords []*WorkflowRecord
	err := s.db.Where("status = ? AND task_id IN (?)", WorkflowStatusRunning, taskIds).Find(&workflowRecords).Error
	return len(workflowRecords) > 0, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetInstanceByWorkflowID(workflowID uint) (*Instance, error) {
	query := `
SELECT instances.id ,instances.maintenance_period
FROM workflows AS w
LEFT JOIN workflow_records AS wr ON wr.id = w.workflow_record_id
LEFT JOIN tasks ON tasks.id = wr.task_id
LEFT JOIN instances ON instances.id = tasks.instance_id
WHERE 
w.id = ?
LIMIT 1`
	instance := &Instance{}
	err := s.db.Raw(query, workflowID).Scan(instance).Error
	if err != nil {
		return nil, errors.ConnectStorageErrWrapper(err)
	}
	return instance, err
}

// GetWorkFlowStepIdsHasAudit 返回走完所有审核流程的workflow_steps的id
// 返回以workflow_record_id为分组的倒数第二条记录的workflow_steps.id
// 如果存在多个工单审核流程，workflow_record_id为分组的倒数第二条记录仍然是判断审核流程是否结束的依据
// 如果不存在工单审核流程，LIMIT 1 offset 1 会将workflow过滤掉
// 每个workflow_record_id对应一个workflows表中的一条记录，返回的id数组可以作为工单数量统计的依据
func (s *Storage) GetWorkFlowStepIdsHasAudit() ([]uint, error) {
	workFlowStepsByIndexAndState, err := s.GetWorkFlowReverseStepsByIndexAndState(1, WorkflowStepStateApprove)
	if err != nil {
		return nil, errors.ConnectStorageErrWrapper(err)
	}

	ids := make([]uint, 0)
	for _, workflowStep := range workFlowStepsByIndexAndState {
		ids = append(ids, workflowStep.ID)
	}

	return ids, nil
}

func (s *Storage) GetDurationMinHasAudit(ids []uint) (int, error) {
	type minStruct struct {
		Min int `json:"min"`
	}

	var result minStruct
	err := s.db.Model(&Workflow{}).
		Select("sum(timestampdiff(minute, workflows.created_at, workflow_steps.operate_at)) as min").
		Joins("LEFT JOIN workflow_steps ON workflow_steps.workflow_record_id = workflows.workflow_record_id").
		Where("workflow_steps.id IN (?)", ids).Scan(&result).Error

	return result.Min, errors.ConnectStorageErrWrapper(err)
}

// WorkFlowStepsBO BO是business object的缩写，表示业务对象
type WorkFlowStepsBO struct {
	ID         uint
	OperateAt  *time.Time
	WorkflowId uint
}

// GetWorkFlowReverseStepsByIndexAndState 返回以workflow_id为分组的倒数第index个记录
func (s *Storage) GetWorkFlowReverseStepsByIndexAndState(index int, state string) ([]*WorkFlowStepsBO, error) {
	query := fmt.Sprintf(`SELECT id,operate_at,workflow_id
FROM workflow_steps a
WHERE a.id =
      (SELECT id
       FROM workflow_steps
       WHERE workflow_id = a.workflow_id
       ORDER BY id desc
       limit 1 offset %d)
  and a.state = '%s';`, index, state)

	workflowStepsBO := make([]*WorkFlowStepsBO, 0)
	return workflowStepsBO, s.db.Raw(query).Scan(&workflowStepsBO).Error
}

func (s *Storage) GetWorkflowCountByStepType(stepTypes []string) (int, error) {
	if len(stepTypes) == 0 {
		return 0, nil
	}

	var count int
	err := s.db.Table("workflows").
		Joins("left join workflow_records on workflows.workflow_record_id = workflow_records.id").
		Joins("left join workflow_steps on workflow_records.current_workflow_step_id = workflow_steps.id").
		Joins("left join workflow_step_templates on workflow_steps.workflow_step_template_id = workflow_step_templates.id ").
		Where("workflow_step_templates.type in (?)", stepTypes).
		Count(&count).Error

	return count, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetWorkflowCountByStatus(status []string) (int, error) {
	if len(status) == 0 {
		return 0, nil
	}

	var count int
	err := s.db.Table("workflows").
		Joins("left join workflow_records on workflows.workflow_record_id = workflow_records.id").
		Where("workflow_records.status in (?)", status).
		Count(&count).Error

	return count, errors.New(errors.ConnectStorageError, err)
}

// GetApprovedWorkflowCount 将会返回未被回收且审核流程全部通过的工单数, 包括上线成功(失败)的工单和等待上线的工单, 不包括关闭的工单
func (s *Storage) GetApprovedWorkflowCount() (int, error) {
	query := `
	select count(1) as count
from workflows 
left join workflow_records on workflows.workflow_record_id = workflow_records.id
left join workflow_steps on workflow_records.current_workflow_step_id = workflow_steps.id 
left join workflow_step_templates on workflow_steps.workflow_step_template_id = workflow_step_templates.id 
where 
workflow_records.status = 'finished'
or
workflow_step_templates.type = 'sql_execute';
`
	var count = struct {
		Count int `json:"count"`
	}{}
	return count.Count, errors.New(errors.ConnectStorageError, s.db.Raw(query).Scan(&count).Error)
}

func (s *Storage) GetAllWorkflowCount() (int, error) {
	var count int
	return count, errors.New(errors.ConnectStorageError, s.db.Model(&Workflow{}).Count(&count).Error)
}

func (s *Storage) GetWorkflowCountByTaskStatus(status []string) (int, error) {
	if len(status) == 0 {
		return 0, nil
	}

	var count int
	err := s.db.Table("workflows").
		Joins("left join workflow_records on workflows.workflow_record_id = workflow_records.id").
		Joins("left join tasks on workflow_records.task_id = tasks.id").
		Where("tasks.status in (?)", status).
		Count(&count).Error

	return count, errors.New(errors.ConnectStorageError, err)
}

func (s *Storage) GetWorkFlowCountBetweenStartTimeAndEndTime(startTime, endTime time.Time) (int64, error) {
	var count int64
	return count, s.db.Model(&Workflow{}).Where("created_at BETWEEN ? and ?", startTime, endTime).Count(&count).Error
}
