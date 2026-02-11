import java.util.Arrays;

public class ArraysSortTest {
    public static void main(String[] args) {
        // Sort int array
        int[] nums = {5, 3, 8, 1, 4, 2, 7, 6};
        Arrays.sort(nums);
        for (int i = 0; i < nums.length; i++) {
            System.out.println(nums[i]);
        }
        // 1 2 3 4 5 6 7 8

        // Binary search
        int idx = Arrays.binarySearch(nums, 5);
        System.out.println(idx); // 4

        // Sort String array
        String[] names = {"Charlie", "Alice", "Bob"};
        Arrays.sort(names);
        for (int i = 0; i < names.length; i++) {
            System.out.println(names[i]);
        }
        // Alice Bob Charlie
    }
}
