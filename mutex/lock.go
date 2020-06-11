/*************************************************************************
 * Copyright (C) 2016-2019 PDX Technologies, Inc. All Rights Reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * @Time   : 2020/6/11 10:55 上午
 * @Author : liangc
 *************************************************************************/

package mutex

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"sync"
	"time"
)

var ns_notfound error = errors.New("Namespace not registed")

// 关于 key 的互斥锁
type KeyMutex struct {
	reglock *sync.Map
	timeout time.Duration
	kcache  *lru.Cache
}

func NewKeyMutex(timeout time.Duration) *KeyMutex {
	cache, _ := lru.New(1024)
	return &KeyMutex{
		reglock: new(sync.Map),
		timeout: timeout,
		kcache:  cache,
	}
}

func (k *KeyMutex) Regist(namespace string) {
	k.reglock.Store(namespace, make(chan struct{}))
}

func (k *KeyMutex) Lock(namespace, key string) (err error) {
	_, ok := k.reglock.Load(namespace)
	if ok {
		v, _ := k.reglock.LoadOrStore(k.hash(namespace, key), make(chan struct{}, 1))
		t := time.NewTimer(k.timeout)
		defer func() {
			t.Stop()
			if ee := recover(); ee != nil {
				err = fmt.Errorf("take lock fail , lost stream : %s@%s : %v", namespace, key, ee)
			}
		}()
		select {
		case v.(chan struct{}) <- struct{}{}:
		case <-t.C:
			err = fmt.Errorf("take lock timeout : %s@%s", namespace, key)
		}
		return
	}
	return ns_notfound
}

func (k *KeyMutex) Unlock(namespace, key string) (err error) {
	_, ok := k.reglock.Load(namespace)
	if ok {
		v, ok := k.reglock.Load(k.hash(namespace, key))
		if ok {
			t := time.NewTimer(k.timeout)
			defer func() {
				t.Stop()
				if recover() != nil {
					err = fmt.Errorf("release lock fail , lost stream : %s@%s", namespace, key)
				}
			}()
			select {
			case <-v.(chan struct{}):
				return nil
			case <-t.C:
				return fmt.Errorf("release lock timeout : %s@%s", namespace, key)
			}
		}
		return
	}
	return ns_notfound
}

// 清除一个旧的锁，会释放一批阻塞的还未超时的 lock 请求，
// 释放以后会在 ns 上产生新的锁,如果 timeout 时间很长，
// 需要提前解除阻塞，可以使用 clean 方法
func (k *KeyMutex) Clean(namespace, key string) {
	if key == "" || namespace == "" {
		return
	}
	if v, ok := k.reglock.Load(k.hash(namespace, key)); ok {
		k.reglock.Delete(k.hash(namespace, key))
		defer func() {
			if r := recover(); r != nil {
				// ignoe
			}
		}()
		close(v.(chan struct{}))
	}
}

func (k *KeyMutex) hash(namespace, key string) string {
	v, ok := k.kcache.Get(namespace + key)
	if ok {
		return v.(string)
	}
	s1 := sha1.New()
	s1.Write([]byte(namespace + key))
	buf := s1.Sum(nil)
	hash := hex.EncodeToString(buf)
	k.kcache.Add(namespace+key, hash)
	return hash
}
