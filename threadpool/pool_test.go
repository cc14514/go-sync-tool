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
 * @Time   : 2020/6/11 10:54 上午
 * @Author : liangc
 *************************************************************************/

package threadpool

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSpawn(t *testing.T) {
	wg := Spawn(10, func(i int) {
		fmt.Println(i)
		time.Sleep(1 * time.Second)
	})
	s := time.Now()
	wg.Wait()
	e := time.Since(s)
	fmt.Println(e)
}

func TestAsyncRunner_Apply(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	runner := NewAsyncRunner(ctx, 3, 6)
	go func() {
		for i := 0; i < 12; i++ {
			runner.Apply(func(ctx context.Context, args []interface{}) {
				i := args[0].(int)
				fmt.Println(ctx.Value("tn"), "AAAAAAAAAAA", i, runner.Size())
				time.Sleep(1 * time.Second)
			}, i)
		}
	}()

	go func() {
		for i := 0; i < 10; i++ {
			runner.Apply(func(ctx context.Context, args []interface{}) {
				i := args[0].(int)
				fmt.Println("tn", ctx.Value("tn"), "BBBBBBBBBB", i, "pool-size", runner.Size())
			}, i)
		}
		time.Sleep(3 * time.Second)
		for i := 0; i < 5; i++ {
			time.Sleep(1 * time.Second)
			runner.Apply(func(ctx context.Context, args []interface{}) {
				i := args[0].(int)
				fmt.Println("tn", ctx.Value("tn"), "CCCCCCCCC", i, "pool-size", runner.Size())
			}, i)
		}
		time.Sleep(3 * time.Second)
	}()
	runner.WaitClose()
	fmt.Println("ttl", time.Since(now), runner.Size())
}
