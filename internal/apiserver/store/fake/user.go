// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package fake

import (
	"strings"

	v1 "github.com/marmotedu/api/apiserver/v1"
	"github.com/marmotedu/component-base/pkg/fields"
	metav1 "github.com/marmotedu/component-base/pkg/meta/v1"
	"github.com/marmotedu/component-base/pkg/util/stringutil"
	"github.com/marmotedu/errors"
	"github.com/marmotedu/iam/internal/apiserver/store"
	"github.com/marmotedu/iam/internal/pkg/code"
	"github.com/marmotedu/iam/internal/pkg/util/gormutil"
	reflectutil "github.com/marmotedu/iam/internal/pkg/util/reflect"
)

type users struct {
	ds *datastore
}

func newUsers(ds *datastore) *users {
	return &users{ds}
}

// Create creates a new user account.
func (u *users) Create(user *v1.User, opts metav1.CreateOptions) error {
	u.ds.Lock()
	defer u.ds.Unlock()

	for _, u := range u.ds.users {
		if u.Name == user.Name {
			return errors.WithCode(code.ErrUserAlreadyExist, "record already exist")
		}
	}

	if len(u.ds.users) > 0 {
		user.ID = u.ds.users[len(u.ds.users)-1].ID + 1
	}
	u.ds.users = append(u.ds.users, user)

	return nil
}

// Update updates an user account information.
func (u *users) Update(user *v1.User, opts metav1.UpdateOptions) error {
	u.ds.Lock()
	defer u.ds.Unlock()

	for _, u := range u.ds.users {
		if u.Name == user.Name {
			if _, err := reflectutil.CopyObj(user, u, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

// Delete deletes the user by the user identifier.
func (u *users) Delete(username string, opts metav1.DeleteOptions) error {
	u.ds.Lock()
	defer u.ds.Unlock()

	// delete related policy first
	if err := store.Client().Policies().DeleteByUser(username, opts); err != nil {
		return err
	}

	users := u.ds.users
	u.ds.users = make([]*v1.User, 0)
	for _, user := range users {
		if user.Name == username {
			continue
		}

		u.ds.users = append(u.ds.users, user)
	}

	return nil
}

// DeleteCollection batch deletes the users.
func (u *users) DeleteCollection(usernames []string, opts metav1.DeleteOptions) error {
	u.ds.Lock()
	defer u.ds.Unlock()

	// delete related policy first
	if err := store.Client().Policies().DeleteCollectionByUser(usernames, opts); err != nil {
		return err
	}

	users := u.ds.users
	u.ds.users = make([]*v1.User, 0)
	for _, user := range users {
		if stringutil.StringIn(user.Name, usernames) {
			continue
		}

		u.ds.users = append(u.ds.users, user)
	}

	return nil
}

// Get return an user by the user identifier.
func (u *users) Get(username string, opts metav1.GetOptions) (*v1.User, error) {
	u.ds.Lock()
	defer u.ds.Unlock()

	for _, u := range u.ds.users {
		if u.Name == username {
			return u, nil
		}
	}

	return nil, errors.WithCode(code.ErrUserNotFound, "record not found")
}

// List return all users.
func (u *users) List(opts metav1.ListOptions) (*v1.UserList, error) {
	u.ds.Lock()
	defer u.ds.Unlock()

	ol := gormutil.Unpointer(opts.Offset, opts.Limit)
	selector, _ := fields.ParseSelector(opts.FieldSelector)
	username, _ := selector.RequiresExactMatch("name")

	users := make([]*v1.User, 0)
	var i int64 = 0
	for _, user := range u.ds.users {
		if i == ol.Limit {
			break
		}
		if !strings.Contains(user.Name, username) {
			continue
		}
		users = append(users, user)
		i++
	}

	return &v1.UserList{
		ListMeta: metav1.ListMeta{
			TotalCount: uint64(len(u.ds.users)),
		},
		Items: users,
	}, nil
}