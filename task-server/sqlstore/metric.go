package sqlstore

import (
	"fmt"
	"strings"
	"time"

	"github.com/raintank/raintank-apps/task-server/model"
)

func ValidateMetrics(orgId int64, metrics map[string]int64) error {
	sess, err := newSession(false, "metric")
	if err != nil {
		return err
	}
	return validateMetrics(sess, orgId, metrics)
}

func validateMetrics(sess *session, orgId int64, metrics map[string]int64) error {
	for namespace, ver := range metrics {
		// for SQL "like" query, "%" is the wildcard character.
		ns := strings.Replace(namespace, "*", "%", -1)
		mQuery := &model.GetMetricsQuery{
			Namespace: ns,
			OrgId:     orgId,
			Version:   ver,
		}
		matches, err := getMetrics(sess, mQuery)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return fmt.Errorf("no matching metric found.")
		}
		//Use the latest version available.
		if ver == 0 {
			for _, m := range matches {
				if m.Version > ver {
					ver = m.Version
				}
			}
			metrics[namespace] = ver
		}
	}
	return nil
}

func GetMetrics(query *model.GetMetricsQuery) ([]*model.Metric, error) {
	sess, err := newSession(false, "metric")
	if err != nil {
		return nil, err
	}
	return getMetrics(sess, query)
}

func getMetrics(sess *session, query *model.GetMetricsQuery) ([]*model.Metric, error) {
	metrics := make([]*model.Metric, 0)
	sess.Where("(public=1 OR org_id = ?)", query.OrgId)
	if query.Namespace != "" {
		sess.And("namespace like ?", query.Namespace)
	}
	if query.Version != 0 {
		sess.And("version = ?", query.Version)
	}
	if query.OrderBy == "" {
		query.OrderBy = "namespace"
	}
	if query.Limit == 0 {
		query.Limit = 50
	}
	if query.Page == 0 {
		query.Page = 1
	}
	sess.Asc(query.OrderBy).Limit(query.Limit, (query.Page-1)*query.Limit)
	err := sess.Find(&metrics)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func GetMetricById(id string, orgId int64) (*model.Metric, error) {
	sess, err := newSession(false, "metric")
	if err != nil {
		return nil, err
	}

	return getMetricById(sess, id, orgId)
}

func getMetricById(sess *session, id string, orgId int64) (*model.Metric, error) {
	m := &model.Metric{}
	exists, err := sess.Where("(public=1 OR org_id = ?) AND id=?", orgId, id).Get(m)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return m, nil
}

func AddMissingMetricsForAgent(a *model.AgentDTO, m []*model.Metric) error {
	sess, err := newSession(true, "metric")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = addMissingMetricsForAgent(sess, a, m); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func addMissingMetricsForAgent(sess *session, agent *model.AgentDTO, metrics []*model.Metric) error {
	existing, err := getAgentMetrics(sess, agent)
	if err != nil {
		return err
	}
	existingMap := make(map[string]*model.Metric)
	seenMap := make(map[string]*model.Metric)

	agentMetrics := make([]*model.AgentMetric, 0)
	for _, m := range existing {
		key := fmt.Sprintf("%s:%d", m.Namespace, m.Version)
		existingMap[key] = m
	}
	for _, m := range metrics {
		key := fmt.Sprintf("%s:%d", m.Namespace, m.Version)
		seenMap[key] = m
		if e, ok := existingMap[key]; ok {
			if e.Public != m.Public {
				// public attribute has changed. need to update
				_, err := sess.Id(e.Id).Update(m)
				if err != nil {
					return err
				}
			}
		} else {
			agentMetrics = append(agentMetrics, &model.AgentMetric{
				AgentId:   agent.Id,
				Namespace: m.Namespace,
				Version:   m.Version,
				Created:   time.Now(),
			})
		}
	}
	for key, m := range existingMap {
		if _, ok := seenMap[key]; !ok {
			// need to delete agent_metric association.
			rawSql := "DELETE from agent_metric where agent_id=? and namespace=? and version=?"
			if _, err := sess.Exec(rawSql, agent.Id, m.Namespace, m.Version); err != nil {
				return err
			}
		}
	}

	if len(agentMetrics) > 0 {
		if _, err := sess.Insert(agentMetrics); err != nil {
			return err
		}

		sess.Table("agent_metric")
		sess.Join("LEFT", "metric", "agent_metric.namespace = metric.namespace AND agent_metric.version = metric.version")
		sess.Where("agent_metric.agent_id=?", agent.Id).And("metric.id IS NULL")
		sess.Cols("`agent_metric`.*")
		needed := make([]model.AgentMetric, 0)
		if err := sess.Find(&needed); err != nil {
			return err
		}
		if len(needed) > 0 {
			inserts := make([]*model.Metric, 0)
			for _, m := range needed {
				key := fmt.Sprintf("%s:%d", m.Namespace, m.Version)
				if metric, ok := seenMap[key]; ok {
					inserts = append(inserts, metric)
				}
			}
			if len(inserts) > 0 {
				if _, err := sess.Insert(inserts); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func GetAgentMetrics(agent *model.AgentDTO) ([]*model.Metric, error) {
	sess, err := newSession(true, "metric")
	if err != nil {
		return nil, err
	}
	defer sess.Cleanup()
	metrics, err := getAgentMetrics(sess, agent)
	if err != nil {
		return nil, err
	}
	sess.Complete()
	return metrics, nil
}

func getAgentMetrics(sess *session, agent *model.AgentDTO) ([]*model.Metric, error) {
	metrics := make([]*model.Metric, 0)
	sess.Table("metric")
	sess.Join("INNER", "agent_metric", "metric.namespace = agent_metric.namespace AND metric.version = agent_metric.version")
	sess.Where("agent_metric.agent_id=?", agent.Id)
	sess.Cols("`metric`.*")
	err := sess.Find(&metrics)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}
