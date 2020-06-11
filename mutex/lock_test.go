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
 * @Time   : 2020/6/11 10:58 上午
 * @Author : liangc
 *************************************************************************/

package mutex

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	ns := "foobar"
	key := "hello"
	km := NewKeyMutex(7 * time.Second)
	km.Regist(ns)
	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			fmt.Println(j, "lock-try")
			if err := km.Lock(ns, key); err != nil {
				fmt.Println(j, "lock-fail", err)
				return
			}
			fmt.Println(j, "lock-success")
			<-time.After(1 * time.Second)
			fmt.Println(j, "unlock-try")
			if err := km.Unlock(ns, key); err != nil {
				fmt.Println(j, "unlock-fail", err)
				return
			}
			fmt.Println(j, "unlock-success")
		}(i)
		if i == 6 {
			km.Clean(ns, key)
			fmt.Println(i, "clean")
		}
	}
	wg.Wait()
	h := km.hash("/premsg/1.0.0", "16Uiu2HAm39zRzVr5JK6P1WCba7ew8L5CBT4r5e3wcZ8V2zQRvWSM")
	fmt.Println(h)

}
